package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/sirupsen/logrus"
	
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
)

type RepositoryIntegrationTestSuite struct {
	IntegrationTestSuite
	repo repositories.UserRepository
}

func (s *RepositoryIntegrationTestSuite) SetupSuite() {
	// Panggil parent SetUp Container dkk.
	s.IntegrationTestSuite.SetupSuite()

	// Inject Real Repository dengan Real Postgres Container DB.
	queries := db.New(s.DB)
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	s.repo = repositories.NewUserRepository(queries, logger)
}

func (s *RepositoryIntegrationTestSuite) TestIntegration_CreateUser_Success() {
	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	userParams := &db.CreateUserParams{
		ID:          uuid.New(),
		Name:        "Budi Santoso",
		Username:    "budisantoso",
		Email:       "budi@test.com",
		Password:    "Bcrypt$10$hashedpassword!",
		PhoneNumber: "08123456789",
		Address:     "Jl Sudirman No. 1",
		Role:        "customer",
	}

	res, err := s.repo.CreateUser(ctx, userParams)

	require.NoError(s.T(), err, "Harusnya sukses insert ke tabel users sungguhan")
	require.NotNil(s.T(), res)
	assert.Equal(s.T(), userParams.ID, res.ID)
	assert.Equal(s.T(), "budi@test.com", res.Email)
}

func (s *RepositoryIntegrationTestSuite) TestIntegration_CreateUser_DuplicateConstraint() {
	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	userParams := &db.CreateUserParams{
		ID:          uuid.New(),
		Name:        "Duplicate Name",
		Username:    "budisantoso", // Sama persis dengan user 1
		Email:       "unique@email.com",
		Password:    "hashed",
	}

	// Buat manual user pertama
	_, err := s.repo.CreateUser(ctx, userParams)
	require.NoError(s.T(), err)

	// User kedua dengan nama ID baru, TAPI username sama
	userParams2 := *userParams
	userParams2.ID = uuid.New()
	userParams2.Email = "baru@email.com"

	_, err = s.repo.CreateUser(ctx, &userParams2)
	require.Error(s.T(), err, "Harusnya gagal akibat Unique Constraint DB postgresql")
	
	// Message error Postgre untuk duplicate constraint: SQLSTATE 23505
	assert.Contains(s.T(), err.Error(), "duplicate key value violates unique constraint")
}

func (s *RepositoryIntegrationTestSuite) TestIntegration_ReadUpdateDelete_Flow() {
	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	// 1. Create
	id := uuid.New()
	userParams := &db.CreateUserParams{
		ID:          id,
		Name:        "Flow User",
		Username:    "flowuser",
		Email:       "flow@test.com",
		Password:    "hash",
		PhoneNumber: "12",
		Address:     "12",
		Role:        "customer",
	}
	_, err := s.repo.CreateUser(ctx, userParams)
	require.NoError(s.T(), err)

	// 2. Read (GetUserByID)
	user, err := s.repo.GetUserByID(ctx, id)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "flow@test.com", user.Email)

	// 3. Update
	updateParams := &db.UpdateUserParams{
		ID:          id,
		Name:        "Flow User Updated",
		Username:    "flowupdated",
		Email:       "flowupdated@test.com",
		Password:    "hash2",
		PhoneNumber: "123",
		Address:     "123",
	}
	updatedUser, err := s.repo.UpdateUser(ctx, updateParams)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "Flow User Updated", updatedUser.Name)

	// 4. Delete (Soft Delete usually or hard delete varying by repo)
	deletedUser, err := s.repo.DeleteUser(ctx, id)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), deletedUser.DeletedAt) // if soft deleted
}

func TestRepositoryIntegrationSuite(t *testing.T) {
	// Integrasi seluruh Lifecycle (TearDown/Setup) dalam 1 Test Tree Runner
	suite.Run(t, new(RepositoryIntegrationTestSuite))
}
