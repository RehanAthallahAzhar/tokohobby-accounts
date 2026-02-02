package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging/rabbitmq"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/helpers"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"
)

type UserHandler struct {
	UserRepo         repositories.UserRepository
	UserService      services.UserService
	TokenService     token.TokenService
	JWTBlacklistRepo repositories.JWTBlacklistRepository
	RefreshTokenRepo repositories.RefreshTokenRepository
	EventPublisher   *rabbitmq.EventPublisher
	log              *logrus.Logger
}

func NewHandler(
	userRepo repositories.UserRepository,
	userService services.UserService,
	tokenService token.TokenService,
	jwtBlacklistRepo repositories.JWTBlacklistRepository,
	refreshTokenRepo repositories.RefreshTokenRepository,
	log *logrus.Logger,
) *UserHandler {
	return &UserHandler{
		UserRepo:         userRepo,
		UserService:      userService,
		TokenService:     tokenService,
		JWTBlacklistRepo: jwtBlacklistRepo,
		RefreshTokenRepo: refreshTokenRepo,
		log:              log,
	}
}

func (h *UserHandler) RegisterUser(c echo.Context) error {
	ctx := c.Request().Context()

	var req models.UserRegisterRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
	}

	userSvc, err := h.UserService.Register(ctx, &req)
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return respondSuccess(c, http.StatusCreated, MsgUserCreated, toUserResponse(userSvc))
}

func (h *UserHandler) Login(c echo.Context) error {
	ctx := c.Request().Context()

	var req models.UserLoginRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
	}

	// Extract activity metadata from HTTP request
	metadata := &services.ActivityMetadata{
		SessionID: c.Request().Header.Get("X-Session-Id"),
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}
	// Fallback to request ID if no session ID
	if metadata.SessionID == "" {
		metadata.SessionID = c.Request().Header.Get("X-Request-Id")
	}

	userSvc, err := h.UserService.Login(ctx, &req, metadata)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	// 15 minutes access token
	accessToken, err := h.TokenService.GenerateAccessToken(ctx, userSvc)
	if err != nil {
		h.log.WithError(err).Error("Failed to generate Access Token")
		return respondError(c, http.StatusInternalServerError, apperrors.ErrFailedToGenerateToken)
	}

	// 7 days refresh token
	refreshToken, err := h.TokenService.GenerateRefreshToken(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to generate Refresh Token")
		return respondError(c, http.StatusInternalServerError, apperrors.ErrFailedToGenerateToken)
	}

	// Store Refresh Token in Redis (e.g. 7 days)
	err = h.RefreshTokenRepo.StoreRefreshToken(ctx, userSvc.ID.String(), refreshToken, 7*24*time.Hour)
	if err != nil {
		h.log.WithError(err).Error("Failed to store Refresh Token")
		return respondError(c, http.StatusInternalServerError, fmt.Errorf("failed to create session"))
	}

	res := toUserResponse(userSvc)
	res.Token = accessToken
	res.RefreshToken = refreshToken

	return respondSuccess(c, http.StatusOK, MsgLogin, res)
}

func (h *UserHandler) RefreshSession(c echo.Context) error {
	ctx := c.Request().Context()

	var req models.RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
	}

	userIDStr, err := h.RefreshTokenRepo.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil || userIDStr == "" {
		h.log.Warnf("Invalid or expired refresh token attempt: %v", err)
		return respondError(c, http.StatusUnauthorized, fmt.Errorf("invalid or expired refresh token"))
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Errorf("Invalid user ID format in session: %v", err)
		return respondError(c, http.StatusInternalServerError, fmt.Errorf("internal session error"))
	}

	userSvc, err := h.UserService.GetUserByID(ctx, userID)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, fmt.Errorf("user not found"))
	}

	// Generate NEW Access Token
	newAccessToken, err := h.TokenService.GenerateAccessToken(ctx, userSvc)
	if err != nil {
		h.log.WithError(err).Error("Failed to generate Access Token during refresh")
		return respondError(c, http.StatusInternalServerError, apperrors.ErrFailedToGenerateToken)
	}

	// Generate NEW Refresh Token
	newRefreshToken, _ := h.TokenService.GenerateRefreshToken(ctx)

	// Revoke old Refresh Token
	h.RefreshTokenRepo.RevokeRefreshToken(ctx, req.RefreshToken)

	// Store new Refresh Token
	h.RefreshTokenRepo.StoreRefreshToken(ctx, userIDStr, newRefreshToken, 7*24*time.Hour)

	return respondSuccess(c, http.StatusOK, "Session refreshed", map[string]string{
		"token": newAccessToken,
	})
}

func (h *UserHandler) Logout(c echo.Context) error {
	ctx := c.Request().Context()

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
	}

	// Optional: Revoke Refresh Token if provided in body?
	// Since we can't extract Refresh Token from Access Token easily without DB lookup,
	// Client *should* send the refresh token to revoke it.
	// But standard logout often just kills access token.
	// If the client deletes the refresh token on their side, it's effectively gone.
	// Server-side revocation requires the client to send the refresh token to be blacklisted.

	// Let's check if client sent Refresh Token in body for revocation
	var req models.RefreshTokenRequest
	if err := c.Bind(&req); err == nil && req.RefreshToken != "" {
		_ = h.RefreshTokenRepo.RevokeRefreshToken(ctx, req.RefreshToken)
	}

	if err := h.UserService.Logout(ctx, authHeader); err != nil {
		return h.handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgLogout, nil)
}

func (h *UserHandler) GetAllUsers(c echo.Context) error {
	ctx := c.Request().Context()

	res, err := h.UserService.GetAllUsers(ctx)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUsersRetrieved, toUserResponses(res))
}

func (h *UserHandler) GetUserByID(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
	}

	res, err := h.UserService.GetUserByID(ctx, id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUserRetrieved, toUserResponse(res))
}

func (h *UserHandler) GetUserProfile(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
	}

	res, err := h.UserService.GetUserByID(ctx, id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUserRetrieved, toUserResponse(res))
}

func (h *UserHandler) UpdateUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
	}

	var req models.UserUpdateRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
	}

	res, err := h.UserService.UpdateUser(ctx, id, &req)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUserUpdated, toUserResponse(res))
}

func (h *UserHandler) DeleteUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := helpers.GetIDFromPathParam(c, "id")
	if err != nil {
		return respondError(c, http.StatusBadRequest, err)
	}

	res, err := h.UserService.DeleteUser(ctx, id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUserDeleted, toUserResponse(res))
}

// ------- HELPERS -------
func toUserResponse(user *entities.User) *models.UserResponse {
	return &models.UserResponse{
		Id:          user.ID,
		Name:        user.Name,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		Address:     user.Address,
		PhoneNumber: user.PhoneNumber,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
	}
}

func toUserResponses(users []entities.User) []models.UserResponse {
	var res []models.UserResponse
	for _, user := range users {
		res = append(res, *toUserResponse(&user))
	}
	return res
}
