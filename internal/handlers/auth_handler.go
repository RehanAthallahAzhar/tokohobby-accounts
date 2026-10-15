package handlers

import (
	"log/slog"
	"net/http"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging/rabbitmq"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	contextpkg "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/context"
	appErrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	AuthService      services.AuthService
	UserService      services.UserService
	JWTBlacklistRepo repositories.JWTBlacklistRepository
	EventPublisher   *rabbitmq.EventPublisher
	log              *slog.Logger
}

func NewAuthHandler(
	authService services.AuthService,
	userService services.UserService,
	jwtBlacklistRepo repositories.JWTBlacklistRepository,
	log *slog.Logger,
) *AuthHandler {
	return &AuthHandler{
		AuthService:      authService,
		UserService:      userService,
		JWTBlacklistRepo: jwtBlacklistRepo,
		log:              log,
	}
}

// Register godoc
// @Summary Registrasi User Baru
// @Description Mendaftarkan pengguna baru ke dalam sistem TokoHobby
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.UserRegisterRequest true "Payload Registrasi"
// @Success 201 {object} models.SuccessResponse{data=models.UserResponse} "User Created"
// @Failure 400 {object} models.ErrorResponse "Invalid Request Payload / Validation Error"
// @Failure 409 {object} models.ErrorResponse "Email or Username already exists"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /api/v1/accounts/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	ctx := c.Request().Context()

	var req models.UserRegisterRequest
	if err := c.Bind(&req); err != nil {
		return handleError(c, http.StatusBadRequest, appErrors.ErrInvalidRequestPayload)
	}

	userSvc, err := h.AuthService.Register(ctx, &req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusCreated, MsgUserCreated, toUserResponse(userSvc))
}

// Login godoc
// @Summary Login User
// @Description Melakukan login ke sistem TokoHobby
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.UserLoginRequest true "Payload Login"
// @Success 200 {object} models.SuccessResponse{data=models.UserAuthResult} "Login Success"
// @Failure 400 {object} models.ErrorResponse "Invalid Request Payload / Validation Error"
// @Failure 401 {object} models.ErrorResponse "Invalid Credentials"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /api/v1/accounts/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	ctx := c.Request().Context()

	var req models.UserLoginRequest
	if err := c.Bind(&req); err != nil {
		return handleError(c, http.StatusBadRequest, appErrors.ErrInvalidRequestPayload)
	}

	sessionID := c.Request().Header.Get("X-Session-Id")
	if sessionID == "" {
		sessionID = c.Response().Header().Get("X-Request-Id")
	}

	metadata := &models.ActivityMetadata{
		SessionID: sessionID,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}

	authResult, err := h.AuthService.Login(ctx, &req, metadata)
	if err != nil {
		return handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgLogin, authResult)
}

// Logout godoc
// @Summary Logout User
// @Description Memvalidasi dan merrevok (membina blacklist) JWT Access Token pengguna di Redis agar tidak dapat digunakan kembali.
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "Logout successful"
// @Failure 400 {object} models.ErrorResponse "Invalid Token Format"
// @Failure 401 {object} models.ErrorResponse "Unauthorized / Invalid Token / Expired Token"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /api/v1/accounts/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	ctx := c.Request().Context()

	authHeader, err := contextpkg.GetAuthorizationHeaderFromContext(ctx)
	if err != nil {
		return handleError(c, http.StatusBadRequest, appErrors.ErrInvalidTokenFormat)
	}

	err = h.AuthService.Logout(ctx, authHeader)
	if err != nil {
		return handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgLogout, nil)
}

// RefreshSession godoc
// @Summary Refresh Session
// @Description Refresh user session berdasarkan old refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RefreshSessionRequest true "Payload Refresh Session"
// @Success 200 {object} models.SuccessResponse{data=models.GenerateTokenResult} "Refresh Session Success"
// @Failure 400 {object} models.ErrorResponse "Invalid Request Payload / Validation Error"
// @Failure 401 {object} models.ErrorResponse "Invalid Session Error"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /api/v1/accounts/refresh [post]
func (h *AuthHandler) RefreshSession(c echo.Context) error {
	ctx := c.Request().Context()

	var req models.RefreshSessionRequest
	if err := c.Bind(&req); err != nil {
		return handleError(c, http.StatusBadRequest, appErrors.ErrInvalidRequestPayload)
	}

	res, err := h.AuthService.RefreshSession(ctx, req.RefreshToken)
	if err != nil {
		return handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgLogin, res)
}
