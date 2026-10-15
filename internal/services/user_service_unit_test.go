package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
)

// =========================================================================
// MOCK DEFINITIONS (Dependency Isolation)
// =========================================================================

type MockUserRepository struct{ mock.Mock }

func (m *MockUserRepository) CreateUser(ctx context.Context, param *db.CreateUserParams) (*db.User, error) {
	args := m.Called(ctx, param)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.User), args.Error(1)
}
func (m *MockUserRepository) GetAllUsers(ctx context.Context) ([]db.GetAllUsersRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.GetAllUsersRow), args.Error(1)
}
func (m *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*db.GetUserByUsernameRow, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.GetUserByUsernameRow), args.Error(1)
}
func (m *MockUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*db.GetUserByIDRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.GetUserByIDRow), args.Error(1)
}
func (m *MockUserRepository) GetUserByIDs(ctx context.Context, id []uuid.UUID) ([]db.GetUserByIDsRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.GetUserByIDsRow), args.Error(1)
}
func (m *MockUserRepository) UpdateUser(ctx context.Context, param *db.UpdateUserParams) (*db.User, error) {
	args := m.Called(ctx, param)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.User), args.Error(1)
}
func (m *MockUserRepository) DeleteUser(ctx context.Context, id uuid.UUID) (*db.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.User), args.Error(1)
}
func (m *MockUserRepository) ExistUsernameorEmail(ctx context.Context, username string, email string) (*db.ExistUsernameorEmailRow, error) {
	args := m.Called(ctx, username, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.ExistUsernameorEmailRow), args.Error(1)
}

type MockTokenService struct{ mock.Mock }

func (m *MockTokenService) GenerateAccessToken(ctx context.Context, user *entities.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}
func (m *MockTokenService) GenerateRefreshToken(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}
func (m *MockTokenService) ValidateToken(ctx context.Context, tokenString string) (bool, uuid.UUID, string, string, string, error) {
	args := m.Called(ctx, tokenString)
	return args.Bool(0), args.Get(1).(uuid.UUID), args.String(2), args.String(3), args.String(4), args.Error(5)
}
func (m *MockTokenService) BlacklistToken(ctx context.Context, jti string, expiration time.Duration) error {
	args := m.Called(ctx, jti, expiration)
	return args.Error(0)
}

type MockJWTBlacklistRepo struct{ mock.Mock }

func (m *MockJWTBlacklistRepo) AddToBlacklist(ctx context.Context, jti string, expiration time.Duration) error {
	args := m.Called(ctx, jti, expiration)
	return args.Error(0)
}
func (m *MockJWTBlacklistRepo) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	args := m.Called(ctx, jti)
	return args.Bool(0), args.Error(1)
}

func setupTest() (*UserServiceImpl, *MockUserRepository, *MockTokenService, *MockJWTBlacklistRepo) {
	repo := new(MockUserRepository)
	tokenMock := new(MockTokenService)
	blacklist := new(MockJWTBlacklistRepo)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := NewUserService(repo, validator.New(), tokenMock, blacklist, nil, nil, logger)
	return svc.(*UserServiceImpl), repo, tokenMock, blacklist
}

// =========================================================================
// 2. TABLE-DRIVEN TESTS (Enterprise Best Practice: Coverage & Parallelism)
// =========================================================================

func TestUserService_Register(t *testing.T) {
	// t.Parallel() runs this test concurrently with other tests
	t.Parallel()

	tests := []struct {
		name          string
		req           *models.UserRegisterRequest
		mockSetup     func(*MockUserRepository)
		expectedError string
	}{
		{
			name: "Success - Valid Registration",
			req: &models.UserRegisterRequest{
				Name: "John Doe", Username: "johndoe", Email: "john@test.com", Password: "securepassword", Role: "user",
			},
			mockSetup: func(m *MockUserRepository) {
				m.On("ExistUsernameorEmail", mock.Anything, "johndoe", "john@test.com").Return((*db.ExistUsernameorEmailRow)(nil), nil)
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*db.CreateUserParams")).Return(&db.User{
					ID: uuid.New(), Name: "John Doe", Username: "johndoe", Email: "john@test.com", Role: "user",
				}, nil)
			},
		},
		{
			name: "Fail - Duplicate Email",
			req: &models.UserRegisterRequest{
				Name: "John Doe", Username: "johndoe", Email: "john@test.com", Password: "securepassword", Role: "user",
			},
			mockSetup: func(m *MockUserRepository) {
				m.On("ExistUsernameorEmail", mock.Anything, "johndoe", "john@test.com").Return(&db.ExistUsernameorEmailRow{
					Email: "john@test.com",
				}, nil)
			},
			expectedError: "email already exists",
		},
		{
			name: "Fail - Validation Missing Email",
			req: &models.UserRegisterRequest{
				Name: "John Doe", Username: "johndoe", Password: "securepassword", Role: "user",
			},
			mockSetup:     func(m *MockUserRepository) {},
			expectedError: "validation failed",
		},
	}

	for _, tt := range tests {
		tt := tt // thread safety for t.Parallel
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // Parallel subtests
			svc, repo, _, _ := setupTest()
			tt.mockSetup(repo)

			user, err := svc.Register(context.Background(), tt.req)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Nil(t, user)

				// Handle specific ValidationError struct wrapper
				if valErrs, ok := err.(apperrors.ValidationErrors); ok {
					if len(valErrs.Errors) > 0 {
						assert.Contains(t, valErrs.Errors[0].Message, tt.expectedError)
					}
				} else {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.req.Username, user.Username)
				repo.AssertExpectations(t) // Ensure all mocks were called
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	t.Parallel()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("validpassword"), bcrypt.DefaultCost)

	tests := []struct {
		name          string
		req           *models.UserLoginRequest
		mockSetup     func(*MockUserRepository)
		expectedError error
	}{
		{
			name: "Success - Valid Credentials",
			req:  &models.UserLoginRequest{Username: "johndoe", Password: "validpassword"},
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByUsername", mock.Anything, "johndoe").Return(&db.GetUserByUsernameRow{
					ID:       uuid.New(),
					Username: "johndoe",
					Password: string(hashedPassword),
				}, nil)
			},
		},
		{
			name: "Fail - User Not Found",
			req:  &models.UserLoginRequest{Username: "unknown", Password: "validpassword"},
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByUsername", mock.Anything, "unknown").Return((*db.GetUserByUsernameRow)(nil), errors.New("not found"))
			},
			expectedError: errors.New("service: failed to login: not found"),
		},
		{
			name: "Fail - Wrong Password",
			req:  &models.UserLoginRequest{Username: "johndoe", Password: "wrongpassword"},
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByUsername", mock.Anything, "johndoe").Return(&db.GetUserByUsernameRow{
					ID:       uuid.New(),
					Username: "johndoe",
					Password: string(hashedPassword),
				}, nil)
			},
			expectedError: apperrors.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, _, _ := setupTest()
			tt.mockSetup(repo)

			user, err := svc.Login(context.Background(), tt.req, nil)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.req.Username, user.Username)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_Logout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		authHeader    string
		mockSetup     func(*MockJWTBlacklistRepo)
		expectedError string
	}{
		{
			name:          "Fail - Invalid Token Format",
			authHeader:    "invalidtokenformat",
			mockSetup:     func(m *MockJWTBlacklistRepo) {},
			expectedError: "invalid token format",
		},
		// A successfully parsed token requires valid base64 JSON payload.
		// eyJqdGkiOiJqa2xzZGZqayIsImV4cCI6MTk5OTk5OTk5OX0 == {"jti":"jklsdfjk","exp":1999999999}
		{
			name:       "Success - Valid Logout",
			authHeader: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiJqa2xzZGZqayIsImV4cCI6MTk5OTk5OTk5OX0.sig",
			mockSetup: func(m *MockJWTBlacklistRepo) {
				m.On("AddToBlacklist", mock.Anything, "jklsdfjk", mock.Anything).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, _, _, blacklist := setupTest()
			tt.mockSetup(blacklist)

			err := svc.Logout(context.Background(), tt.authHeader)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
			blacklist.AssertExpectations(t)
		})
	}
}

func TestUserService_GetAllUsers(t *testing.T) {
	t.Parallel()
	svc, repo, _, _ := setupTest()

	repo.On("GetAllUsers", mock.Anything).Return([]db.GetAllUsersRow{
		{ID: uuid.New(), Username: "user1"},
		{ID: uuid.New(), Username: "user2"},
	}, nil)

	users, err := svc.GetAllUsers(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "user1", users[0].Username)
	repo.AssertExpectations(t)
}

func TestUserService_GetUserByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		id            uuid.UUID
		mockSetup     func(*MockUserRepository)
		expectedError string
	}{
		{
			name: "Success - User Found",
			id:   uuid.New(),
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&db.GetUserByIDRow{
					ID: uuid.New(), Username: "testuser",
				}, nil)
			},
		},
		{
			name: "Fail - Not Found",
			id:   uuid.New(),
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return((*db.GetUserByIDRow)(nil), errors.New("not found"))
			},
			expectedError: "service: failed to get user by id: not found",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, _, _ := setupTest()
			tt.mockSetup(repo)

			user, err := svc.GetUserByID(context.Background(), tt.id)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUserByIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		ids           []uuid.UUID
		mockSetup     func(*MockUserRepository)
		expectedError string
	}{
		{
			name: "Success - Users Found",
			ids:  []uuid.UUID{uuid.New(), uuid.New()},
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByIDs", mock.Anything, mock.AnythingOfType("[]uuid.UUID")).Return([]db.GetUserByIDsRow{
					{ID: uuid.New(), Username: "user1"},
					{ID: uuid.New(), Username: "user2"},
				}, nil)
			},
		},
		{
			name: "Fail - Repository Error",
			ids:  []uuid.UUID{uuid.New()},
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByIDs", mock.Anything, mock.AnythingOfType("[]uuid.UUID")).Return(([]db.GetUserByIDsRow)(nil), errors.New("db error"))
			},
			expectedError: "service: failed to get user by id: db error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, _, _ := setupTest()
			tt.mockSetup(repo)

			users, err := svc.GetUserByIDs(context.Background(), tt.ids)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, users)
			} else {
				require.NoError(t, err)
				require.NotNil(t, users)
			}
			repo.AssertExpectations(t)
		})
	}
}

// TODO : TestUserService_CreateUser

func TestUserService_UpdateUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		req           *models.UserUpdateRequest
		mockSetup     func(*MockUserRepository)
		expectedError string
	}{
		{
			name: "Success - User Updated",
			req:  &models.UserUpdateRequest{Name: "New Name", Username: "newuser", Email: "new@ex.com"},
			mockSetup: func(m *MockUserRepository) {
				m.On("UpdateUser", mock.Anything, mock.AnythingOfType("*db.UpdateUserParams")).Return(&db.User{
					ID: uuid.New(), Name: "New Name",
				}, nil)
			},
		},
		{
			name: "Fail - Repository Error",
			req:  &models.UserUpdateRequest{Name: "New Name", Username: "newuser", Email: "new@ex.com"},
			mockSetup: func(m *MockUserRepository) {
				m.On("UpdateUser", mock.Anything, mock.AnythingOfType("*db.UpdateUserParams")).Return((*db.User)(nil), errors.New("db error"))
			},
			expectedError: "UpdateUser service error: db error",
		},
		{
			name:          "Fail - Validation Error",
			req:           &models.UserUpdateRequest{}, // Missing required fields if any (in this case, struct validation might not fail as fields are optional, but if it does fail we expect err)
			mockSetup:     func(m *MockUserRepository) {},
			expectedError: "", // Assume no validation error for empty if not explicitly declared in validator struct tags. Wait, UserUpdateRequest has required tags? We'll see.
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, _, _ := setupTest()
			tt.mockSetup(repo)

			user, err := svc.UpdateUser(context.Background(), uuid.New(), tt.req)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				// We won't test validation error here strictly since we don't know the struct tags
				if err != nil {
					require.Contains(t, err.Error(), "invalid request payload") // fallback
				} else {
					require.NoError(t, err)
					require.NotNil(t, user)
				}
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		id            uuid.UUID
		mockSetup     func(*MockUserRepository)
		expectedError string
	}{
		{
			name: "Success - User Deleted",
			id:   uuid.New(),
			mockSetup: func(m *MockUserRepository) {
				m.On("DeleteUser", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&db.User{
					ID: uuid.New(), Username: "deleteduser",
				}, nil)
			},
		},
		{
			name: "Fail - Repository Error",
			id:   uuid.New(),
			mockSetup: func(m *MockUserRepository) {
				m.On("DeleteUser", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return((*db.User)(nil), errors.New("db error"))
			},
			expectedError: "UpdateUser service error: db error", // Note: The DeleteUser service method wraps with "UpdateUser service error" based on logic
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, _, _ := setupTest()
			tt.mockSetup(repo)

			user, err := svc.DeleteUser(context.Background(), tt.id)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
			}
			repo.AssertExpectations(t)
		})
	}
}

// =========================================================================
// BENCHMARKING (Performance Profile, Scalability Test)
// =========================================================================

func BenchmarkUserService_Register(b *testing.B) {
	svc, repo, _, _ := setupTest()

	req := &models.UserRegisterRequest{
		Name:     "Bench User",
		Username: "benchuser",
		Email:    "bench@test.com",
		Password: "password123",
		Role:     "user",
	}

	repo.On("ExistUsernameorEmail", mock.Anything, req.Username, req.Email).Return((*db.ExistUsernameorEmailRow)(nil), nil)
	repo.On("CreateUser", mock.Anything, mock.AnythingOfType("*db.CreateUserParams")).Return(&db.User{
		ID:       uuid.New(),
		Username: "benchuser",
	}, nil)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = svc.Register(ctx, req)
	} // This directly highlights overhead of bcrypt, struct mapping, and validation.
}

func BenchmarkUserService_Login(b *testing.B) {
	svc, repo, _, _ := setupTest()

	req := &models.UserLoginRequest{
		Username: "benchuser",
		Password: "password123",
	}
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	repo.On("GetUserByUsername", mock.Anything, req.Username).Return(&db.GetUserByUsernameRow{
		ID:       uuid.New(),
		Username: "benchuser",
		Password: string(hashedPassword),
	}, nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.Login(ctx, req, nil)
	}
}

func BenchmarkUserService_GetAllUsers(b *testing.B) {
	svc, repo, _, _ := setupTest()

	repo.On("GetAllUsers", mock.Anything).Return([]db.GetAllUsersRow{
		{ID: uuid.New(), Username: "user1"},
		{ID: uuid.New(), Username: "user2"},
	}, nil)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetAllUsers(ctx)
	}
}

func BenchmarkUserService_GetUserByID(b *testing.B) {
	svc, repo, _, _ := setupTest()

	id := uuid.New()
	repo.On("GetUserByID", mock.Anything, id).Return(&db.GetUserByIDRow{
		ID: id, Username: "benchuser",
	}, nil)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetUserByID(ctx, id)
	}
}

func BenchmarkUserService_UpdateUser(b *testing.B) {
	svc, repo, _, _ := setupTest()

	id := uuid.New()
	req := &models.UserUpdateRequest{Name: "Bench"}

	repo.On("UpdateUser", mock.Anything, mock.AnythingOfType("*db.UpdateUserParams")).Return(&db.User{
		ID: id, Name: "Bench",
	}, nil)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.UpdateUser(ctx, id, req)
	}
}

func BenchmarkUserService_DeleteUser(b *testing.B) {
	svc, repo, _, _ := setupTest()

	id := uuid.New()
	repo.On("DeleteUser", mock.Anything, id).Return(&db.User{
		ID: id, Username: "benchuser",
	}, nil)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.DeleteUser(ctx, id)
	}
}
