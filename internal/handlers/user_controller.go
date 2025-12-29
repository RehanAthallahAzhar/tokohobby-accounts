package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
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
	log              *logrus.Logger
}

func NewHandler(
	userRepo repositories.UserRepository,
	userService services.UserService,
	tokenService token.TokenService,
	jwtBlacklistRepo repositories.JWTBlacklistRepository,
	log *logrus.Logger,
) *UserHandler {
	return &UserHandler{
		UserRepo:         userRepo,
		UserService:      userService,
		TokenService:     tokenService,
		JWTBlacklistRepo: jwtBlacklistRepo,
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

	userSvc, err := h.UserService.Login(ctx, &req)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	signedToken, err := h.TokenService.GenerateToken(ctx, userSvc)
	if err != nil {
		h.log.WithError(err).Error("Failed to generate JWT")
		return respondError(c, http.StatusInternalServerError, apperrors.ErrFailedToGenerateToken)
	}

	res := toUserResponse(userSvc)
	res.Token = signedToken

	return respondSuccess(c, http.StatusOK, MsgLogin, res)
}

func (h *UserHandler) Logout(c echo.Context) error {
	ctx := c.Request().Context()

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
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

func (h *UserHandler) GetUserById(c echo.Context) error {
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
