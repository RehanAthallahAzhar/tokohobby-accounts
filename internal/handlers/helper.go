package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	apperrors "github.com/RehanAthallahAzhar/shopeezy-accounts/internal/pkg/errors"

	"github.com/RehanAthallahAzhar/shopeezy-accounts/internal/models"
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
		log.Println("DEBUG: c.Get('userID') hasilnya nil.")
		return uuid.Nil, errors.New("invalid user session: userID is nil in context")
	}

	if id, ok := val.(uuid.UUID); ok {
		return id, nil
	}

	log.Printf("DEBUG: Gagal assertion ke uuid.UUID. Tipe data aktual adalah %T", val)
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
