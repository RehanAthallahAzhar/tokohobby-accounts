package repositories

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
)

// SETUP (Muted Logger, Transient Mock)
func setupRepositoryMockTest() (*sql.DB, sqlmock.Sqlmock, UserRepository) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		panic("failed to create sqlmock: " + err.Error())
	}

	queries := db.New(mockDB)
	repo := NewUserRepository(queries)

	return mockDB, mock, repo
}

func TestUserRepositoryUnit_ExistUsernameorEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		username      string
		email         string
		mockSetup     func(sqlmock.Sqlmock)
		expectedError error
		expectedFound bool
	}{
		{
			name:     "Success - Found existing record",
			username: "johndoe",
			email:    "johndoe@example.com",
			mockSetup: func(m sqlmock.Sqlmock) {
				// Note : Exact matching of the SQL string can be brittle, so we rely on regex escaping
				query := regexp.QuoteMeta(`SELECT username, email FROM users WHERE (username = $1 OR email = $2) AND deleted_at IS NULL LIMIT 1`)
				rows := sqlmock.NewRows([]string{"username", "email"}).AddRow("johndoe", "johndoe@example.com")
				m.ExpectQuery(query).WithArgs("johndoe", "johndoe@example.com").WillReturnRows(rows)
			},
			expectedFound: true,
		},
		{
			name:     "Success - No record found",
			username: "unknown",
			email:    "unknown@example.com",
			mockSetup: func(m sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(`SELECT username, email FROM users WHERE (username = $1 OR email = $2) AND deleted_at IS NULL LIMIT 1`)
				m.ExpectQuery(query).WithArgs("unknown", "unknown@example.com").WillReturnError(sql.ErrNoRows)
			},
			expectedFound: false,
		},
		{
			name:     "Success - No record found",
			username: "unknown",
			email:    "",
			mockSetup: func(m sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(`SELECT username, email FROM users WHERE (username = $1 OR email = $2) AND deleted_at IS NULL LIMIT 1`)
				m.ExpectQuery(query).WithArgs("unknown", "").WillReturnError(sql.ErrNoRows)
			},
			expectedFound: false,
		},
		{
			name:     "Success - No record found",
			username: "",
			email:    "unknown@example.com",
			mockSetup: func(m sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(`SELECT username, email FROM users WHERE (username = $1 OR email = $2) AND deleted_at IS NULL LIMIT 1`)
				m.ExpectQuery(query).WithArgs("", "unknown@example.com").WillReturnError(sql.ErrNoRows)
			},
			expectedFound: false,
		},
		{
			name:     "Success - No record found",
			username: "",
			email:    "",
			mockSetup: func(m sqlmock.Sqlmock) {
				query := regexp.QuoteMeta(`SELECT username, email FROM users WHERE (username = $1 OR email = $2) AND deleted_at IS NULL LIMIT 1`)
				m.ExpectQuery(query).WithArgs("", "").WillReturnError(sql.ErrNoRows)
			},
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, mock, repo := setupRepositoryMockTest()
			t.Cleanup(func() {
				assert.NoError(t, mockDB.Close())
			})

			tt.mockSetup(mock)
			ctx := context.Background()

			result, err := repo.ExistUsernameorEmail(ctx, tt.username, tt.email)

			if tt.expectedError != nil {
				require.Error(t, err)
			} else {
				if !tt.expectedFound {
					assert.Nil(t, result)
				} else {
					require.NoError(t, err)
					require.NotNil(t, result)
					assert.Equal(t, tt.username, result.Username)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepositoryUnit_GetAllUsers(t *testing.T) {
	t.Parallel()
	mockDB, mock, repo := setupRepositoryMockTest()
	t.Cleanup(func() {
		assert.NoError(t, mockDB.Close())
	})

	query := regexp.QuoteMeta(`SELECT id, "name", username, email, "password",phone_number, "address", "role", created_at, updated_at FROM users WHERE deleted_at IS NULL`)

	rows := sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at"}).
		AddRow(uuid.New(), "User 1", "user1", "user1@ex.com", "hash", "123", "add", "user", time.Now(), time.Now()).
		AddRow(uuid.New(), "User 2", "user2", "user2@ex.com", "hash", "123", "add", "user", time.Now(), time.Now())

	mock.ExpectQuery(query).WillReturnRows(rows)

	users, err := repo.GetAllUsers(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "user1", users[0].Username)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepositoryUnit_CreateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		param         *db.CreateUserParams
		mockSetup     func(sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name: "Success - Created User",
			param: &db.CreateUserParams{
				ID: uuid.New(), Name: "Test User", Username: "testuser", Email: "test@ex.com",
			},
			mockSetup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("(?i)INSERT INTO users").WillReturnRows(
					sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at", "deleted_at"}).
						AddRow(uuid.New(), "Test User", "testuser", "test@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now(), nil),
				)
			},
		},
		{
			name:          "Fail - Nil Param",
			param:         nil,
			mockSetup:     func(m sqlmock.Sqlmock) {},
			expectedError: "invalid query parameter",
		},
		{
			name: "Fail - DB Error",
			param: &db.CreateUserParams{
				ID: uuid.New(), Name: "Test User", Username: "testuser", Email: "test@ex.com",
			},
			mockSetup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("(?i)INSERT INTO users").WillReturnError(sql.ErrConnDone)
			},
			expectedError: "failed to create user",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, mock, repo := setupRepositoryMockTest()
			t.Cleanup(func() {
				assert.NoError(t, mockDB.Close())
			})

			tt.mockSetup(mock)
			ctx := context.Background()

			res, err := repo.CreateUser(ctx, tt.param)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, tt.param.Username, res.Username)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepositoryUnit_GetUserByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            uuid.UUID
		mockSetup     func(sqlmock.Sqlmock, uuid.UUID)
		expectedError string
	}{
		{
			name: "Success - User Found",
			id:   uuid.New(),
			mockSetup: func(m sqlmock.Sqlmock, id uuid.UUID) {
				m.ExpectQuery("(?i)SELECT .* FROM users WHERE id = \\$1").WithArgs(id).WillReturnRows(
					sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at"}).
						AddRow(id, "Test User", "testuser", "test@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now()),
				)
			},
		},
		{
			name: "Fail - Not Found",
			id:   uuid.New(),
			mockSetup: func(m sqlmock.Sqlmock, id uuid.UUID) {
				m.ExpectQuery("(?i)SELECT .* FROM users WHERE id = \\$1").WithArgs(id).WillReturnError(sql.ErrNoRows)
			},
			expectedError: "failed to get user by id",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, mock, repo := setupRepositoryMockTest()
			t.Cleanup(func() {
				assert.NoError(t, mockDB.Close())
			})

			tt.mockSetup(mock, tt.id)
			ctx := context.Background()

			res, err := repo.GetUserByID(ctx, tt.id)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, tt.id, res.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepositoryUnit_GetUserByIDs(t *testing.T) {
	t.Parallel()

	id1, id2 := uuid.New(), uuid.New()

	tests := []struct {
		name          string
		ids           []uuid.UUID
		mockSetup     func(sqlmock.Sqlmock, []uuid.UUID)
		expectedError string
	}{
		{
			name: "Success - Users Found",
			ids:  []uuid.UUID{id1, id2},
			mockSetup: func(m sqlmock.Sqlmock, ids []uuid.UUID) {
				m.ExpectQuery("(?i)SELECT .* FROM users").WillReturnRows(
					sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at"}).
						AddRow(ids[0], "Test User 1", "user1", "user1@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now()).
						AddRow(ids[1], "Test User 2", "user2", "user2@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now()),
				)
			},
		},
		{
			name: "Fail - DB Error",
			ids:  []uuid.UUID{id1},
			mockSetup: func(m sqlmock.Sqlmock, ids []uuid.UUID) {
				m.ExpectQuery("(?i)SELECT .* FROM users").WillReturnError(sql.ErrConnDone)
			},
			expectedError: "failed to get user by id",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, mock, repo := setupRepositoryMockTest()
			t.Cleanup(func() {
				assert.NoError(t, mockDB.Close())
			})

			tt.mockSetup(mock, tt.ids)
			ctx := context.Background()

			res, err := repo.GetUserByIDs(ctx, tt.ids)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Len(t, res, len(tt.ids))
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepositoryUnit_UpdateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		param         *db.UpdateUserParams
		mockSetup     func(sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name: "Success - Updated User",
			param: &db.UpdateUserParams{
				ID: uuid.New(), Name: "Updated Name",
			},
			mockSetup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("(?i)UPDATE users SET").WillReturnRows(
					sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at", "deleted_at"}).
						AddRow(uuid.New(), "Updated Name", "user1", "user1@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now(), nil),
				)
			},
		},
		{
			name:  "Fail - DB Error",
			param: &db.UpdateUserParams{ID: uuid.New()},
			mockSetup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("(?i)UPDATE users SET").WillReturnError(sql.ErrNoRows)
			},
			expectedError: "failed to update user",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, mock, repo := setupRepositoryMockTest()
			t.Cleanup(func() {
				assert.NoError(t, mockDB.Close())
			})

			tt.mockSetup(mock)
			ctx := context.Background()

			res, err := repo.UpdateUser(ctx, tt.param)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepositoryUnit_DeleteUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            uuid.UUID
		mockSetup     func(sqlmock.Sqlmock, uuid.UUID)
		expectedError string
	}{
		{
			name: "Success - Deleted User",
			id:   uuid.New(),
			mockSetup: func(m sqlmock.Sqlmock, id uuid.UUID) {
				m.ExpectQuery("(?i)(UPDATE users SET deleted_at|DELETE FROM users)").WithArgs(id).WillReturnRows(
					sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at", "deleted_at"}).
						AddRow(id, "Deleted Name", "user1", "user1@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now(), time.Now()),
				)
			},
		},
		{
			name: "Fail - DB Error",
			id:   uuid.New(),
			mockSetup: func(m sqlmock.Sqlmock, id uuid.UUID) {
				m.ExpectQuery("(?i)(UPDATE users SET deleted_at|DELETE FROM users)").WithArgs(id).WillReturnError(sql.ErrNoRows)
			},
			expectedError: "failed to delete user",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, mock, repo := setupRepositoryMockTest()
			t.Cleanup(func() {
				assert.NoError(t, mockDB.Close())
			})

			tt.mockSetup(mock, tt.id)
			ctx := context.Background()

			res, err := repo.DeleteUser(ctx, tt.id)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, tt.id, res.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Fixed Benchmark using setup in loop:
func BenchmarkUserRepository_GetUserByUsername(b *testing.B) {
	mockDB, mock, repo := setupRepositoryMockTest()
	defer func() {
		_ = mockDB.Close()
	}()

	username := "benchuser"
	query := regexp.QuoteMeta(`SELECT id, "name", username, email, "password",phone_number, "address", "role", created_at, updated_at FROM users WHERE username = $1 AND deleted_at IS NULL`)

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery(query).WithArgs(username).WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at"}).
				AddRow(uuid.New(), "Bench", username, "bench@ex.com", "pass", "123", "addr", "user", time.Now(), time.Now()),
		)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.GetUserByUsername(ctx, username)
	}
}

// -------------------------------------------------------------------------
// BENCHMARKS (Scalability Testing)
// cmd : go test -bench=^BenchmarkUserRepository_CreateUser$ -benchmem -benchtime=100x ./internal/repositories (benchtime can be 100x, 1000x, etc or 30s, 60s, etc)
// -------------------------------------------------------------------------

func BenchmarkUserRepository_CreateUser(b *testing.B) {
	mockDB, mock, repo := setupRepositoryMockTest()
	defer func() {
		_ = mockDB.Close()
	}()

	param := &db.CreateUserParams{
		ID: uuid.New(), Name: "Bench", Username: "bench", Email: "bench@ex.com",
	}

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("(?i)INSERT INTO users").WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at", "deleted_at"}).
				AddRow(param.ID, "Test User", param.Username, "test@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now(), nil),
		)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.CreateUser(ctx, param)
	}
}

func BenchmarkUserRepository_GetAllUsers(b *testing.B) {
	mockDB, mock, repo := setupRepositoryMockTest()
	defer func() {
		_ = mockDB.Close()
	}()

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("(?i)SELECT .* FROM users").WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at", "deleted_at"}).
				AddRow(uuid.New(), "User1", "user1", "u1@e.com", "x", "1", "a", "u", time.Now(), time.Now(), nil).
				AddRow(uuid.New(), "User2", "user2", "u2@e.com", "x", "1", "a", "u", time.Now(), time.Now(), nil),
		)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.GetAllUsers(ctx)
	}
}

func BenchmarkUserRepository_GetUserByID(b *testing.B) {
	mockDB, mock, repo := setupRepositoryMockTest()
	defer func() {
		_ = mockDB.Close()
	}()

	id := uuid.New()

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("(?i)SELECT .* FROM users WHERE id = \\$1").WithArgs(id).WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at"}).
				AddRow(id, "Test User", "testuser", "test@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now()),
		)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.GetUserByID(ctx, id)
	}
}

func BenchmarkUserRepository_UpdateUser(b *testing.B) {
	mockDB, mock, repo := setupRepositoryMockTest()
	defer func() {
		_ = mockDB.Close()
	}()

	param := &db.UpdateUserParams{ID: uuid.New(), Name: "Bench"}

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("(?i)UPDATE users SET").WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at", "deleted_at"}).
				AddRow(param.ID, param.Name, "testuser", "test@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now(), nil),
		)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.UpdateUser(ctx, param)
	}
}

func BenchmarkUserRepository_DeleteUser(b *testing.B) {
	mockDB, mock, repo := setupRepositoryMockTest()
	defer func() {
		_ = mockDB.Close()
	}()

	id := uuid.New()

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("(?i)(UPDATE users SET deleted_at|DELETE FROM users)").WithArgs(id).WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "username", "email", "password", "phone_number", "address", "role", "created_at", "updated_at", "deleted_at"}).
				AddRow(id, "User1", "testuser", "test@ex.com", "pwd", "123", "addr", "user", time.Now(), time.Now(), time.Now()),
		)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = repo.DeleteUser(ctx, id)
	}
}
