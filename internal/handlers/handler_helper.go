package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
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

func handleServiceError(c echo.Context, err error) error {
	if err == nil {
		return nil
	}

	// Context Errors
	if errors.Is(err, context.Canceled) {
		// 499 Client Closed Request (Client Disconnect)
		return handleError(c, 499, apperrors.ErrClientCanceled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		// 504 Gateway Timeout / Request Timeout
		return handleError(c, http.StatusGatewayTimeout, apperrors.ErrRequestTimeout)
	}

	// validation errors
	var valErrs apperrors.ValidationErrors
	if errors.As(err, &valErrs) {
		fieldErrs := make([]models.FieldErrorResponse, 0, len(valErrs.Errors))

		for _, ve := range valErrs.Errors {
			fieldErrs = append(fieldErrs, models.FieldErrorResponse{
				Field: ve.Field,
				Msg:   ve.Message,
			})
		}

		return c.JSON(http.StatusBadRequest, models.ErrorDetailsResponse{
			Error:   "Validation failed",
			Details: fieldErrs,
		})
	}

	var customAppErr *apperrors.AppError
	if errors.As(err, &customAppErr) {
		// If you use custom AppError struct that carries its own StatusCode
		return handleError(c, customAppErr.Code, customAppErr)
	}

	// Sentinel Domain Errors
	switch {
	// 404 NOT FOUND
	case errors.Is(err, apperrors.ErrUserNotFound),
		errors.Is(err, apperrors.ErrResourceNotFound):
		return handleError(c, http.StatusNotFound, err)

	// 409 CONFLICT / DUPLICATE
	case errors.Is(err, apperrors.ErrUserAlreadyExists),
		errors.Is(err, apperrors.ErrEmailAlreadyExists),
		errors.Is(err, apperrors.ErrUsernameAlreadyExists):
		return handleError(c, http.StatusConflict, err)

	// 401 UNAUTHORIZED
	case errors.Is(err, apperrors.ErrInvalidCredentials),
		errors.Is(err, apperrors.ErrInvalidToken),
		errors.Is(err, apperrors.ErrExpiredToken),
		errors.Is(err, apperrors.ErrMissingJTI):
		return handleError(c, http.StatusUnauthorized, err)

	// 403 FORBIDDEN
	case errors.Is(err, apperrors.ErrForbidden),
		errors.Is(err, apperrors.ErrAccountDisabled):
		return handleError(c, http.StatusForbidden, err)

	// 400 BAD REQUEST
	case errors.Is(err, apperrors.ErrInvalidRequestPayload),
		errors.Is(err, apperrors.ErrInvalidTokenFormat):
		return handleError(c, http.StatusBadRequest, err)

	// 422 UNPROCESSABLE ENTITY
	case errors.Is(err, apperrors.ErrUnprocessableEntity):
		return handleError(c, http.StatusUnprocessableEntity, err)

	// 503 SERVICE UNAVAILABLE
	case errors.Is(err, apperrors.ErrDatabaseConnection),
		errors.Is(err, apperrors.ErrRedisConnection),
		errors.Is(err, apperrors.ErrThirdPartyUnavailable):
		return handleError(c, http.StatusServiceUnavailable, apperrors.ErrServiceUnavailable)
	}

	// // grpc
	// if st, ok := status.FromError(err); ok {
	// 	switch st.Code() {
	// 	case codes.NotFound:
	// 		return handleError(c, http.StatusNotFound, apperrors.ErrResourceNotFound)
	// 	case codes.AlreadyExists:
	// 		return handleError(c, http.StatusConflict, apperrors.ErrAlreadyExists)
	// 	case codes.InvalidArgument:
	// 		return handleError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
	// 	case codes.Unauthenticated:
	// 		return handleError(c, http.StatusUnauthorized, apperrors.ErrInvalidToken)
	// 	case codes.PermissionDenied:
	// 		return handleError(c, http.StatusForbidden, apperrors.ErrForbidden)
	// 	case codes.Unavailable:
	// 		return handleError(c, http.StatusServiceUnavailable, apperrors.ErrServiceUnavailable)
	// 	case codes.DeadlineExceeded:
	// 		return handleError(c, http.StatusGatewayTimeout, apperrors.ErrRequestTimeout)
	// 	}
	// }

	// unhandle error
	return handleError(c, http.StatusInternalServerError, apperrors.ErrInternalServerError)
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

func handleError(c echo.Context, statusCode int, appErr error) error {
	ctx := c.Request().Context()
	span := trace.SpanFromContext(ctx)

	// Record error to OpenTelemetry Span
	if span.IsRecording() {
		span.RecordError(appErr)
		span.SetStatus(codes.Error, appErr.Error())
	}
	reqID := c.Response().Header().Get("X-Request-Id")

	// Filter log level to avoid triggering false alerts in production
	if statusCode >= 500 {
		slog.ErrorContext(ctx, appErr.Error(),
			slog.String("request_id", reqID),
			slog.Int("status_code", statusCode),
			slog.Any("error", appErr),
		)
	} else {
		slog.WarnContext(ctx, appErr.Error(),
			slog.String("request_id", reqID),
			slog.Int("status_code", statusCode),
			slog.Any("error", appErr),
		)
	}

	return respondError(c, statusCode, appErr)
}
