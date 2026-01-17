package handlers

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	apperrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
)

// ------- HELPERS -------

const (
	MsgUserRetrieved  = "User retrieved successfully"
	MsgUserCreated    = "User created successfully"
	MsgUserUpdated    = "User updated successfully"
	MsgUserDeleted    = "User deleted successfully"
	MsgUsersRetrieved = "Users retrieved successfully"
	MsgLogin          = "Login successful"
	MsgLogout         = "Logout successful"
)

func extractUserID(c echo.Context) (uuid.UUID, error) {
	val := c.Get("userID")

	if val == nil {
		return uuid.Nil, errors.New("invalid user session: userID is nil in context")
	}

	if id, ok := val.(uuid.UUID); ok {
		return id, nil
	}

	return uuid.Nil, errors.New("invalid user session: userID in context is not of type uuid.UUID")
}

func respondSuccess(c echo.Context, status int, message string, data interface{}) error {
	return c.JSON(status, models.SuccessResponse{
		Message: message,
		Data:    data,
	})
}

func respondError(c echo.Context, status int, err error) error {
	return c.JSON(status, models.ErrorResponse{
		Error: err.Error(),
	})
}

func (h *UserHandler) handleServiceError(c echo.Context, err error) error {
	// Check for structured validation errors (field-level)
	var validationErrs apperrors.ValidationErrors
	if errors.As(err, &validationErrs) {
		return c.JSON(http.StatusBadRequest, validationErrs)
	}

	// validation error
	if errors.Is(err, apperrors.ErrInvalidRequestPayload) {
		return respondError(c, http.StatusBadRequest, err)
	}

	// Authentication & Authorization Errors (401 & 403)
	if errors.Is(err, apperrors.ErrInvalidCredentials) {
		return respondError(c, http.StatusUnauthorized, err)
	}
	if errors.Is(err, apperrors.ErrInvalidToken) {
		return respondError(c, http.StatusUnauthorized, err)
	}
	if errors.Is(err, apperrors.ErrForbidden) {
		return respondError(c, http.StatusForbidden, err)
	}

	// not found
	if errors.Is(err, apperrors.ErrNotFound) {
		return respondError(c, http.StatusNotFound, err)
	}

	// Data Conflict
	if errors.Is(err, apperrors.ErrUserAlreadyExists) {
		return respondError(c, http.StatusConflict, err)
	}
	if errors.Is(err, apperrors.ErrUsernameAlreadyExists) {
		return respondError(c, http.StatusConflict, err)
	}
	if errors.Is(err, apperrors.ErrEmailAlreadyExists) {
		return respondError(c, http.StatusConflict, err)
	}

	// Out of Stock Product
	if errors.Is(err, apperrors.ErrProductOutOfStock) {
		return respondError(c, http.StatusUnprocessableEntity, err)
	}

	h.log.WithFields(logrus.Fields{
		"request_id": c.Response().Header().Get(echo.HeaderXRequestID),
		"error":      err.Error(),
	}).Error("An unexpected internal server error occurred")

	return respondError(c, http.StatusInternalServerError, apperrors.ErrInternalServerError)
}
