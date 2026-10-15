package handlers

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/helpers"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging/rabbitmq"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	contextpkg "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/context"
	appErrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services"
)

type UserHandler struct {
	UserService      services.UserService
	JWTBlacklistRepo repositories.JWTBlacklistRepository
	EventPublisher   *rabbitmq.EventPublisher
	log              *slog.Logger
}

func NewUserHandler(
	userService services.UserService,
	jwtBlacklistRepo repositories.JWTBlacklistRepository,
	log *slog.Logger,
) *UserHandler {
	return &UserHandler{
		UserService:      userService,
		JWTBlacklistRepo: jwtBlacklistRepo,
		log:              log,
	}
}

// GetAllUsers godoc
// @Summary Get All Users
// @Description Mendapatkan semua user dalam sistem TokoHobby
// @Tags Users
// @Produce json
// @Success 200 {object} models.SuccessResponse{data=[]models.UserResponse} "Users Retrieved"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /api/v1/accounts [get]
// @Security BearerAuth
func (h *UserHandler) GetAllUsers(c echo.Context) error {
	ctx := c.Request().Context()

	res, err := h.UserService.GetAllUsers(ctx)
	if err != nil {
		return handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUsersRetrieved, toUserResponses(res))
}

// GetUserByID godoc
// @Summary Get User by ID
// @Description Mendapatkan user berdasarkan ID
// @Tags Users
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Success 200 {object} models.SuccessResponse{data=models.UserResponse} "User Retrieved"
// @Failure 400 {object} models.ErrorResponse "Invalid Request Payload / Validation Error"
// @Failure 404 {object} models.ErrorResponse "User Not Found"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /api/v1/accounts/{id} [get]
// @Security BearerAuth
func (h *UserHandler) GetUserByID(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := helpers.GetIDFromPathParam(c, "id")
	if err != nil {
		return handleError(c, http.StatusBadRequest, appErrors.ErrInvalidRequestPayload)
	}

	res, err := h.UserService.GetUserByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUserRetrieved, toUserResponse(res))
}

// GetUserProfile godoc
// @Summary Get User Profile
// @Description Mendapatkan profil user saat ini
// @Tags Users
// @Produce json
// @Success 200 {object} models.SuccessResponse{data=models.UserResponse} "User Retrieved"
// @Failure 401 {object} models.ErrorResponse "Invalid User Session"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /api/v1/accounts/profile [get]
// @Security BearerAuth
func (h *UserHandler) GetUserProfile(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := contextpkg.GetUserIDFromContext(ctx)
	if err != nil {
		return handleError(c, http.StatusUnauthorized, appErrors.ErrInvalidUserSession)
	}

	res, err := h.UserService.GetUserByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUserRetrieved, toUserResponse(res))
}

// UpdateUser godoc
// @Summary Update User
// @Description Update user berdasarkan ID
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Param request body models.UserUpdateRequest true "Payload Update User"
// @Success 200 {object} models.SuccessResponse{data=models.UserResponse} "User Updated"
// @Failure 400 {object} models.ErrorResponse "Invalid Request Payload / Validation Error"
// @Failure 404 {object} models.ErrorResponse "User Not Found"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /api/v1/accounts/{id} [put]
// @Security BearerAuth
func (h *UserHandler) UpdateUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := contextpkg.GetUserIDFromContext(ctx)
	if err != nil {
		return handleError(c, http.StatusUnauthorized, appErrors.ErrInvalidUserSession)
	}

	var req models.UserUpdateRequest
	if err := c.Bind(&req); err != nil {
		return handleError(c, http.StatusBadRequest, appErrors.ErrInvalidRequestPayload)
	}

	res, err := h.UserService.UpdateUser(ctx, id, &req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUserUpdated, toUserResponse(res))
}

// DeleteUser godoc
// @Summary Delete User
// @Description Delete user berdasarkan ID
// @Tags Users
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Success 200 {object} models.SuccessResponse{data=models.UserResponse} "User Deleted"
// @Failure 400 {object} models.ErrorResponse "Invalid Request Payload / Validation Error"
// @Failure 404 {object} models.ErrorResponse "User Not Found"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /api/v1/accounts/{id} [delete]
// @Security BearerAuth
func (h *UserHandler) DeleteUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := helpers.GetIDFromPathParam(c, "id")
	if err != nil {
		return handleError(c, http.StatusBadRequest, appErrors.ErrInvalidRequestPayload)
	}

	res, err := h.UserService.DeleteUser(ctx, id)
	if err != nil {
		return handleServiceError(c, err)
	}

	return respondSuccess(c, http.StatusOK, MsgUserDeleted, toUserResponse(res))
}
