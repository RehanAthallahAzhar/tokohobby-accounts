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

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	appErrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
)

// =========================================================================
// 1. MOCK DEFINITIONS (Layer Isolation)
// =========================================================================

type MockUserRepository struct {
	mock.Mock
}

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

func (m *MockUserRepository) GetUserByIDs(ctx context.Context, ids []uuid.UUID) ([]db.GetUserByIDsRow, error) {
	args := m.Called(ctx, ids)
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

// Helpers
func setupServiceTest() (*UserServiceImpl, *MockUserRepository) {
	repo := new(MockUserRepository)
	validate := validator.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := NewUserService(repo, validate, nil, nil, nil, nil, nil, logger)
	return svc, repo
}

func stringPtr(s string) *string {
	return &s
}

// =========================================================================
// 2. TABLE-DRIVEN TESTS (Enterprise Coverage)
// =========================================================================

func TestUserService_GetAllUsers(t *testing.T) {
	t.Parallel()

	now := time.Now()
	user1ID := uuid.New()
	user2ID := uuid.New()

	tests := []struct {
		name          string
		mockSetup     func(*MockUserRepository)
		expectedCount int
		expectedError error
	}{
		{
			name: "Success - Multiple Users Retrieved",
			mockSetup: func(m *MockUserRepository) {
				m.On("GetAllUsers", mock.Anything).Return([]db.GetAllUsersRow{
					{
						ID:          user1ID,
						Name:        "User One",
						Username:    "user1",
						Email:       "user1@test.com",
						Role:        "user",
						Address:     "Jl. Mawar",
						PhoneNumber: "08123456789",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
					{
						ID:          user2ID,
						Name:        "User Two",
						Username:    "user2",
						Email:       "user2@test.com",
						Role:        "admin",
						Address:     "Jl. Melati",
						PhoneNumber: "08987654321",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				}, nil)
			},
			expectedCount: 2,
			expectedError: nil,
		},
		{
			name: "Success - Empty Users List",
			mockSetup: func(m *MockUserRepository) {
				m.On("GetAllUsers", mock.Anything).Return([]db.GetAllUsersRow{}, nil)
			},
			expectedCount: 0,
			expectedError: nil,
		},
		{
			name: "Fail - Repository Error",
			mockSetup: func(m *MockUserRepository) {
				m.On("GetAllUsers", mock.Anything).Return(([]db.GetAllUsersRow)(nil), errors.New("database connection failure"))
			},
			expectedCount: 0,
			expectedError: errors.New("database connection failure"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := setupServiceTest()
			tt.mockSetup(repo)

			users, err := svc.GetAllUsers(context.Background())

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
				assert.Nil(t, users)
			} else {
				require.NoError(t, err)
				assert.Len(t, users, tt.expectedCount)
				if tt.expectedCount > 0 {
					assert.Equal(t, user1ID, users[0].ID)
					assert.Equal(t, "user1", users[0].Username)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	now := time.Now()

	tests := []struct {
		name          string
		id            uuid.UUID
		mockSetup     func(*MockUserRepository)
		expectedError error
		validateUser  func(*testing.T, *entities.User)
	}{
		{
			name: "Success - User Found",
			id:   targetID,
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByID", mock.Anything, targetID).Return(&db.GetUserByIDRow{
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
			expectedError: nil,
			validateUser: func(t *testing.T, u *entities.User) {
				require.NotNil(t, u)
				assert.Equal(t, targetID, u.ID)
				assert.Equal(t, "John Doe", u.Name)
				assert.Equal(t, "johndoe", u.Username)
				assert.Equal(t, "john@test.com", u.Email)
			},
		},
		{
			name: "Fail - User Not Found",
			id:   targetID,
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByID", mock.Anything, targetID).Return((*db.GetUserByIDRow)(nil), appErrors.ErrUserNotFound)
			},
			expectedError: appErrors.ErrUserNotFound,
			validateUser: func(t *testing.T, u *entities.User) {
				assert.Nil(t, u)
			},
		},
		{
			name: "Fail - Repository Query Error",
			id:   targetID,
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByID", mock.Anything, targetID).Return((*db.GetUserByIDRow)(nil), errors.New("sql query error"))
			},
			expectedError: errors.New("sql query error"),
			validateUser: func(t *testing.T, u *entities.User) {
				assert.Nil(t, u)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := setupServiceTest()
			tt.mockSetup(repo)

			user, err := svc.GetUserByID(context.Background(), tt.id)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}

			tt.validateUser(t, user)
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUserByIDs(t *testing.T) {
	t.Parallel()

	id1 := uuid.New()
	id2 := uuid.New()
	targetIDs := []uuid.UUID{id1, id2}
	now := time.Now()

	tests := []struct {
		name          string
		ids           []uuid.UUID
		mockSetup     func(*MockUserRepository)
		expectedCount int
		expectedError error
	}{
		{
			name: "Success - Users Found by IDs",
			ids:  targetIDs,
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByIDs", mock.Anything, targetIDs).Return([]db.GetUserByIDsRow{
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
						Role:        "user",
						Address:     "Addr 2",
						PhoneNumber: "082",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				}, nil)
			},
			expectedCount: 2,
			expectedError: nil,
		},
		{
			name: "Success - Empty IDs Returned",
			ids:  []uuid.UUID{},
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByIDs", mock.Anything, []uuid.UUID{}).Return([]db.GetUserByIDsRow{}, nil)
			},
			expectedCount: 0,
			expectedError: nil,
		},
		{
			name: "Fail - Repository Error",
			ids:  targetIDs,
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByIDs", mock.Anything, targetIDs).Return(([]db.GetUserByIDsRow)(nil), errors.New("db connection failure"))
			},
			expectedCount: 0,
			expectedError: errors.New("db connection failure"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := setupServiceTest()
			tt.mockSetup(repo)

			users, err := svc.GetUserByIDs(context.Background(), tt.ids)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
				assert.Nil(t, users)
			} else {
				require.NoError(t, err)
				assert.Len(t, users, tt.expectedCount)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	now := time.Now()

	tests := []struct {
		name          string
		id            uuid.UUID
		req           *models.UserUpdateRequest
		mockSetup     func(*MockUserRepository)
		expectedError string
		validateUser  func(*testing.T, *entities.User)
	}{
		{
			name: "Success - Valid Update With All Fields",
			id:   targetID,
			req: &models.UserUpdateRequest{
				Name:        stringPtr("Updated Name"),
				Address:     stringPtr("Jl. Baru No. 123"),
				PhoneNumber: stringPtr("081234567890"),
			},
			mockSetup: func(m *MockUserRepository) {
				m.On("UpdateUser", mock.Anything, &db.UpdateUserParams{
					ID:          targetID,
					Name:        "Updated Name",
					Address:     "Jl. Baru No. 123",
					PhoneNumber: "081234567890",
				}).Return(&db.User{
					ID:          targetID,
					Name:        "Updated Name",
					Username:    "existinguser",
					Email:       "user@test.com",
					Role:        "user",
					Address:     "Jl. Baru No. 123",
					PhoneNumber: "081234567890",
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil)
			},
			expectedError: "",
			validateUser: func(t *testing.T, u *entities.User) {
				require.NotNil(t, u)
				assert.Equal(t, targetID, u.ID)
				assert.Equal(t, "Updated Name", u.Name)
				assert.Equal(t, "Jl. Baru No. 123", u.Address)
				assert.Equal(t, "081234567890", u.PhoneNumber)
			},
		},
		{
			name: "Fail - Validation Error (Missing Required Name)",
			id:   targetID,
			req: &models.UserUpdateRequest{
				Name:        nil,
				Address:     stringPtr("Jl. Baru"),
				PhoneNumber: stringPtr("081234567890"),
			},
			mockSetup:     func(m *MockUserRepository) {},
			expectedError: "Key: 'UserUpdateRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag",
			validateUser: func(t *testing.T, u *entities.User) {
				assert.Nil(t, u)
			},
		},
		{
			name: "Fail - Repository Update Error",
			id:   targetID,
			req: &models.UserUpdateRequest{
				Name:        stringPtr("Updated Name"),
				Address:     stringPtr("Jl. Baru"),
				PhoneNumber: stringPtr("081234567890"),
			},
			mockSetup: func(m *MockUserRepository) {
				m.On("UpdateUser", mock.Anything, mock.AnythingOfType("*db.UpdateUserParams")).
					Return((*db.User)(nil), errors.New("db error updating user"))
			},
			expectedError: "db error updating user",
			validateUser: func(t *testing.T, u *entities.User) {
				assert.Nil(t, u)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := setupServiceTest()
			tt.mockSetup(repo)

			user, err := svc.UpdateUser(context.Background(), tt.id, tt.req)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			tt.validateUser(t, user)
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	now := time.Now()

	tests := []struct {
		name          string
		id            uuid.UUID
		mockSetup     func(*MockUserRepository)
		expectedError error
		validateUser  func(*testing.T, *entities.User)
	}{
		{
			name: "Success - User Deleted Successfully",
			id:   targetID,
			mockSetup: func(m *MockUserRepository) {
				m.On("DeleteUser", mock.Anything, targetID).Return(&db.User{
					ID:          targetID,
					Name:        "Deleted User",
					Username:    "deleteduser",
					Email:       "deleted@test.com",
					Role:        "user",
					Address:     "Addr",
					PhoneNumber: "081",
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil)
			},
			expectedError: nil,
			validateUser: func(t *testing.T, u *entities.User) {
				require.NotNil(t, u)
				assert.Equal(t, targetID, u.ID)
				assert.Equal(t, "deleteduser", u.Username)
			},
		},
		{
			name: "Fail - User Not Found",
			id:   targetID,
			mockSetup: func(m *MockUserRepository) {
				m.On("DeleteUser", mock.Anything, targetID).Return((*db.User)(nil), appErrors.ErrUserNotFound)
			},
			expectedError: appErrors.ErrUserNotFound,
			validateUser: func(t *testing.T, u *entities.User) {
				assert.Nil(t, u)
			},
		},
		{
			name: "Fail - Repository Error",
			id:   targetID,
			mockSetup: func(m *MockUserRepository) {
				m.On("DeleteUser", mock.Anything, targetID).Return((*db.User)(nil), errors.New("sql execute failure"))
			},
			expectedError: errors.New("sql execute failure"),
			validateUser: func(t *testing.T, u *entities.User) {
				assert.Nil(t, u)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := setupServiceTest()
			tt.mockSetup(repo)

			user, err := svc.DeleteUser(context.Background(), tt.id)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}

			tt.validateUser(t, user)
			repo.AssertExpectations(t)
		})
	}
}

// =========================================================================
// 3. BENCHMARKING (Performance & Memory Allocation Profiling)
// =========================================================================

func BenchmarkUserService_GetAllUsers(b *testing.B) {
	svc, repo := setupServiceTest()
	now := time.Now()

	repo.On("GetAllUsers", mock.Anything).Return([]db.GetAllUsersRow{
		{ID: uuid.New(), Username: "user1", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Username: "user2", CreatedAt: now, UpdatedAt: now},
	}, nil)

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = svc.GetAllUsers(ctx)
	}
}

func BenchmarkUserService_GetUserByID(b *testing.B) {
	svc, repo := setupServiceTest()
	targetID := uuid.New()
	now := time.Now()

	repo.On("GetUserByID", mock.Anything, targetID).Return(&db.GetUserByIDRow{
		ID:        targetID,
		Username:  "benchuser",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = svc.GetUserByID(ctx, targetID)
	}
}

func BenchmarkUserService_GetUserByIDs(b *testing.B) {
	svc, repo := setupServiceTest()
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	now := time.Now()

	repo.On("GetUserByIDs", mock.Anything, ids).Return([]db.GetUserByIDsRow{
		{ID: ids[0], Username: "user1", CreatedAt: now, UpdatedAt: now},
		{ID: ids[1], Username: "user2", CreatedAt: now, UpdatedAt: now},
	}, nil)

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = svc.GetUserByIDs(ctx, ids)
	}
}

func BenchmarkUserService_UpdateUser(b *testing.B) {
	svc, repo := setupServiceTest()
	targetID := uuid.New()
	now := time.Now()

	req := &models.UserUpdateRequest{
		Name:        stringPtr("Bench Name"),
		Address:     stringPtr("Bench Addr"),
		PhoneNumber: stringPtr("081234567890"),
	}

	repo.On("UpdateUser", mock.Anything, mock.AnythingOfType("*db.UpdateUserParams")).Return(&db.User{
		ID:        targetID,
		Name:      "Bench Name",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = svc.UpdateUser(ctx, targetID, req)
	}
}

func BenchmarkUserService_DeleteUser(b *testing.B) {
	svc, repo := setupServiceTest()
	targetID := uuid.New()
	now := time.Now()

	repo.On("DeleteUser", mock.Anything, targetID).Return(&db.User{
		ID:        targetID,
		Username:  "benchuser",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = svc.DeleteUser(ctx, targetID)
	}
}
