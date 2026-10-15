package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	contextpkg "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/context"
	appErrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
)

// =========================================================================
// 1. MOCK DEFINITIONS (Layer Isolation)
// =========================================================================

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetAllUsers(ctx context.Context) ([]entities.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entities.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserService) GetUserByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.User, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entities.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, id uuid.UUID, req *models.UserUpdateRequest) (*entities.User, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

// Helpers
func setupHandlerTest() (*echo.Echo, *UserHandler, *MockUserService) {
	e := echo.New()
	mockSvc := new(MockUserService)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewUserHandler(mockSvc, nil, logger)
	return e, handler, mockSvc
}

func stringPtr(s string) *string {
	return &s
}

// =========================================================================
// 2. TABLE-DRIVEN TESTS (Enterprise API Validation)
// =========================================================================

func TestUserHandler_GetAllUsers(t *testing.T) {
	t.Parallel()

	now := time.Now()
	id1 := uuid.New()
	id2 := uuid.New()

	tests := []struct {
		name           string
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Success - 200 OK with Users",
			mockSetup: func(m *MockUserService) {
				m.On("GetAllUsers", mock.Anything).Return([]entities.User{
					{
						ID:          id1,
						Name:        "User 1",
						Username:    "user1",
						Email:       "u1@test.com",
						Role:        "user",
						Address:     "Addr 1",
						PhoneNumber: "081",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
					{
						ID:          id2,
						Name:        "User 2",
						Username:    "user2",
						Email:       "u2@test.com",
						Role:        "admin",
						Address:     "Addr 2",
						PhoneNumber: "082",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.SuccessResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, MsgUsersRetrieved, resp.Message)

				dataList, ok := resp.Data.([]interface{})
				require.True(t, ok)
				assert.Len(t, dataList, 2)
			},
		},
		{
			name: "Fail - Service 500 Internal Server Error",
			mockSetup: func(m *MockUserService) {
				m.On("GetAllUsers", mock.Anything).Return(nil, errors.New("db query failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInternalServerError.Error(), resp.Error)
			},
		},
		{
			name: "Fail - Service 504 Gateway Timeout",
			mockSetup: func(m *MockUserService) {
				m.On("GetAllUsers", mock.Anything).Return(nil, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusGatewayTimeout,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrRequestTimeout.Error(), resp.Error)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e, handler, mockSvc := setupHandlerTest()
			tt.mockSetup(mockSvc)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetAllUsers(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateBody != nil {
				tt.validateBody(t, rec)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_GetUserByID(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	now := time.Now()

	tests := []struct {
		name           string
		pathParamID    string
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "Success - 200 OK User Found",
			pathParamID: targetID.String(),
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByID", mock.Anything, targetID).Return(&entities.User{
					ID:          targetID,
					Name:        "John Doe",
					Username:    "johndoe",
					Email:       "john@test.com",
					Role:        "user",
					Address:     "Jl. Sudirman",
					PhoneNumber: "081234567890",
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.SuccessResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, MsgUserRetrieved, resp.Message)

				dataMap, ok := resp.Data.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, targetID.String(), dataMap["id"])
				assert.Equal(t, "johndoe", dataMap["username"])
			},
		},
		{
			name:           "Fail - Invalid UUID Path Param 400",
			pathParamID:    "invalid-uuid-string",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInvalidRequestPayload.Error(), resp.Error)
			},
		},
		{
			name:        "Fail - User Not Found 404",
			pathParamID: targetID.String(),
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByID", mock.Anything, targetID).Return(nil, appErrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrUserNotFound.Error(), resp.Error)
			},
		},
		{
			name:        "Fail - Service Database Error 500",
			pathParamID: targetID.String(),
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByID", mock.Anything, targetID).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInternalServerError.Error(), resp.Error)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e, handler, mockSvc := setupHandlerTest()
			tt.mockSetup(mockSvc)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+tt.pathParamID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.pathParamID)

			err := handler.GetUserByID(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateBody != nil {
				tt.validateBody(t, rec)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_GetUserProfile(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	now := time.Now()

	tests := []struct {
		name           string
		setContextID   bool
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "Success - 200 OK Profile Retrieved",
			setContextID: true,
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByID", mock.Anything, userID).Return(&entities.User{
					ID:          userID,
					Name:        "Profile User",
					Username:    "profileuser",
					Email:       "profile@test.com",
					Role:        "user",
					Address:     "Jl. Thamrin",
					PhoneNumber: "081234567890",
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.SuccessResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, MsgUserRetrieved, resp.Message)

				dataMap, ok := resp.Data.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, userID.String(), dataMap["id"])
				assert.Equal(t, "profileuser", dataMap["username"])
			},
		},
		{
			name:           "Fail - Unauthorized 401 Missing Context User ID",
			setContextID:   false,
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInvalidUserSession.Error(), resp.Error)
			},
		},
		{
			name:         "Fail - User Not Found 404",
			setContextID: true,
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByID", mock.Anything, userID).Return(nil, appErrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrUserNotFound.Error(), resp.Error)
			},
		},
		{
			name:         "Fail - Service Database Error 500",
			setContextID: true,
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("db connection failure"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInternalServerError.Error(), resp.Error)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e, handler, mockSvc := setupHandlerTest()
			tt.mockSetup(mockSvc)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/profile", nil)
			if tt.setContextID {
				ctx := contextpkg.SetUserIDToContext(req.Context(), userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetUserProfile(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateBody != nil {
				tt.validateBody(t, rec)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_UpdateUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	now := time.Now()

	tests := []struct {
		name           string
		setContextID   bool
		requestBody    interface{}
		rawBodyString  string
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "Success - 200 OK User Updated",
			setContextID: true,
			requestBody: &models.UserUpdateRequest{
				Name:        stringPtr("Jane Doe"),
				Address:     stringPtr("Jl. Baru No. 1"),
				PhoneNumber: stringPtr("0811111111"),
			},
			mockSetup: func(m *MockUserService) {
				m.On("UpdateUser", mock.Anything, userID, mock.MatchedBy(func(req *models.UserUpdateRequest) bool {
					return req != nil && req.Name != nil && *req.Name == "Jane Doe"
				})).Return(&entities.User{
					ID:          userID,
					Name:        "Jane Doe",
					Username:    "existinguser",
					Email:       "user@test.com",
					Role:        "user",
					Address:     "Jl. Baru No. 1",
					PhoneNumber: "0811111111",
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.SuccessResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, MsgUserUpdated, resp.Message)

				dataMap, ok := resp.Data.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "Jane Doe", dataMap["name"])
			},
		},
		{
			name:         "Fail - Unauthorized 401 Missing Context User ID",
			setContextID: false,
			requestBody: &models.UserUpdateRequest{
				Name: stringPtr("Jane Doe"),
			},
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInvalidUserSession.Error(), resp.Error)
			},
		},
		{
			name:           "Fail - Invalid Request Payload (Malformed JSON) 400",
			setContextID:   true,
			rawBodyString:  `{"name": "broken json`,
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInvalidRequestPayload.Error(), resp.Error)
			},
		},
		{
			name:         "Fail - Validation Errors From Service 400",
			setContextID: true,
			requestBody: &models.UserUpdateRequest{
				Name: nil,
			},
			mockSetup: func(m *MockUserService) {
				m.On("UpdateUser", mock.Anything, userID, mock.AnythingOfType("*models.UserUpdateRequest")).
					Return(nil, appErrors.ValidationErrors{
						Errors: []appErrors.ValidationError{
							{Field: "name", Message: "name is required"},
						},
					})
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorDetailsResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, "Validation failed", resp.Error)
				require.Len(t, resp.Details, 1)
				assert.Equal(t, "name", resp.Details[0].Field)
			},
		},
		{
			name:         "Fail - User Not Found 404",
			setContextID: true,
			requestBody: &models.UserUpdateRequest{
				Name: stringPtr("Jane Doe"),
			},
			mockSetup: func(m *MockUserService) {
				m.On("UpdateUser", mock.Anything, userID, mock.AnythingOfType("*models.UserUpdateRequest")).
					Return(nil, appErrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrUserNotFound.Error(), resp.Error)
			},
		},
		{
			name:         "Fail - Service Internal Error 500",
			setContextID: true,
			requestBody: &models.UserUpdateRequest{
				Name: stringPtr("Jane Doe"),
			},
			mockSetup: func(m *MockUserService) {
				m.On("UpdateUser", mock.Anything, userID, mock.AnythingOfType("*models.UserUpdateRequest")).
					Return(nil, errors.New("database update failure"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInternalServerError.Error(), resp.Error)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e, handler, mockSvc := setupHandlerTest()
			tt.mockSetup(mockSvc)

			var bodyBytes []byte
			if tt.rawBodyString != "" {
				bodyBytes = []byte(tt.rawBodyString)
			} else if tt.requestBody != nil {
				var marshalErr error
				bodyBytes, marshalErr = json.Marshal(tt.requestBody)
				require.NoError(t, marshalErr)
			}

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/accounts", bytes.NewReader(bodyBytes))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			if tt.setContextID {
				ctx := contextpkg.SetUserIDToContext(req.Context(), userID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.UpdateUser(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateBody != nil {
				tt.validateBody(t, rec)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_DeleteUser(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	now := time.Now()

	tests := []struct {
		name           string
		pathParamID    string
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "Success - 200 OK User Deleted",
			pathParamID: targetID.String(),
			mockSetup: func(m *MockUserService) {
				m.On("DeleteUser", mock.Anything, targetID).Return(&entities.User{
					ID:          targetID,
					Name:        "Deleted User",
					Username:    "deleteduser",
					Email:       "del@test.com",
					Role:        "user",
					Address:     "Addr",
					PhoneNumber: "081",
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.SuccessResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, MsgUserDeleted, resp.Message)
			},
		},
		{
			name:           "Fail - Invalid UUID Path Param 400",
			pathParamID:    "invalid-uuid",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInvalidRequestPayload.Error(), resp.Error)
			},
		},
		{
			name:        "Fail - User Not Found 404",
			pathParamID: targetID.String(),
			mockSetup: func(m *MockUserService) {
				m.On("DeleteUser", mock.Anything, targetID).Return(nil, appErrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrUserNotFound.Error(), resp.Error)
			},
		},
		{
			name:        "Fail - Service Database Error 500",
			pathParamID: targetID.String(),
			mockSetup: func(m *MockUserService) {
				m.On("DeleteUser", mock.Anything, targetID).Return(nil, errors.New("db delete error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp models.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, appErrors.ErrInternalServerError.Error(), resp.Error)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e, handler, mockSvc := setupHandlerTest()
			tt.mockSetup(mockSvc)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/accounts/"+tt.pathParamID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.pathParamID)

			err := handler.DeleteUser(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateBody != nil {
				tt.validateBody(t, rec)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

// =========================================================================
// 3. BENCHMARKING (HTTP Pipeline & Overhead Profiling)
// =========================================================================

func BenchmarkUserHandler_GetAllUsers(b *testing.B) {
	e, handler, mockSvc := setupHandlerTest()
	now := time.Now()

	mockSvc.On("GetAllUsers", mock.Anything).Return([]entities.User{
		{ID: uuid.New(), Username: "user1", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Username: "user2", CreatedAt: now, UpdatedAt: now},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		_ = handler.GetAllUsers(c)
	}
}

func BenchmarkUserHandler_GetUserByID(b *testing.B) {
	e, handler, mockSvc := setupHandlerTest()
	targetID := uuid.New()
	targetIDStr := targetID.String()
	now := time.Now()

	mockSvc.On("GetUserByID", mock.Anything, targetID).Return(&entities.User{
		ID:        targetID,
		Username:  "user1",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+targetIDStr, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(targetIDStr)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		_ = handler.GetUserByID(c)
	}
}

func BenchmarkUserHandler_GetUserProfile(b *testing.B) {
	e, handler, mockSvc := setupHandlerTest()
	userID := uuid.New()
	now := time.Now()

	mockSvc.On("GetUserByID", mock.Anything, userID).Return(&entities.User{
		ID:        userID,
		Username:  "profileuser",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/profile", nil)
	ctx := contextpkg.SetUserIDToContext(req.Context(), userID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		_ = handler.GetUserProfile(c)
	}
}

func BenchmarkUserHandler_UpdateUser(b *testing.B) {
	e, handler, mockSvc := setupHandlerTest()
	userID := uuid.New()
	now := time.Now()

	mockSvc.On("UpdateUser", mock.Anything, userID, mock.AnythingOfType("*models.UserUpdateRequest")).Return(&entities.User{
		ID:        userID,
		Name:      "Updated User",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	payload := models.UserUpdateRequest{
		Name:        stringPtr("Updated User"),
		Address:     stringPtr("Jl. Baru"),
		PhoneNumber: stringPtr("08123456789"),
	}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/accounts", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	ctx := contextpkg.SetUserIDToContext(req.Context(), userID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		rec.Body.Reset()
		_ = handler.UpdateUser(c)
	}
}

func BenchmarkUserHandler_DeleteUser(b *testing.B) {
	e, handler, mockSvc := setupHandlerTest()
	targetID := uuid.New()
	targetIDStr := targetID.String()
	now := time.Now()

	mockSvc.On("DeleteUser", mock.Anything, targetID).Return(&entities.User{
		ID:        targetID,
		Username:  "deleteduser",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/accounts/"+targetIDStr, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(targetIDStr)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		_ = handler.DeleteUser(c)
	}
}
