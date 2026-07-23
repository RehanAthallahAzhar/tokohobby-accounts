package tests

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services"
)

type ServiceIntegrationTestSuite struct {
	IntegrationTestSuite
	userSvc services.UserService
	logger  *logrus.Logger
}

func (s *ServiceIntegrationTestSuite) SetupSuite() {
	s.IntegrationTestSuite.SetupSuite()

	s.logger = logrus.New()
	s.logger.SetLevel(logrus.FatalLevel)

	// Real Repo
	queries := db.New(s.DB)
	userRepo := repositories.NewUserRepository(queries, s.logger)

	// Mocks for Redis/Tokens
	mockTs := new(MockTokenService)
	mockBl := new(MockJWTBlacklistRepo)
	val := validator.New()

	// Real User Service
	s.userSvc = services.NewUserService(userRepo, val, mockTs, mockBl, nil, nil, s.logger)
}

func (s *ServiceIntegrationTestSuite) TestService_Integration_RegisterAndLogin() {
	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	// 1. Register User via Service Layer directly
	registerReq := &models.UserRegisterRequest{
		Name:     "Service Integration",
		Username: "serviceuser",
		Email:    "service@test.com",
		Password: "SecurePassword123!",
	}

	user, err := s.userSvc.Register(ctx, registerReq)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), user)
	assert.Equal(s.T(), "serviceuser", user.Username)

	// 2. Validate missing field triggers validation error, not DB constraint wrapper
	badReq := &models.UserRegisterRequest{
		Username: "bad",
	}
	_, err = s.userSvc.Register(ctx, badReq)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "validation failed")

	// 3. Login Flow via Service Layer
	loginReq := &models.UserLoginRequest{
		Username: "serviceuser",
		Password: "SecurePassword123!",
	}
	meta := &services.ActivityMetadata{IPAddress: "127.0.0.1"}
	
	loggedInUser, err := s.userSvc.Login(ctx, loginReq, meta)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), loggedInUser)
	assert.Equal(s.T(), user.ID, loggedInUser.ID)
}

func (s *ServiceIntegrationTestSuite) TestService_Integration_UpdateDeleteFlow() {
	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	// 1. Register
	req := &models.UserRegisterRequest{
		Name:     "To Delete",
		Username: "todelete",
		Email:    "del@test.com",
		Password: "abc",
	}
	u, err := s.userSvc.Register(ctx, req)
	require.NoError(s.T(), err)

	// 2. Update
	updateReq := &models.UserUpdateRequest{
		Name:     "Updated Name",
		Username: "todelete",
		Email:    "del@test.com",
		Password: "abc",
	}
	u2, err := s.userSvc.UpdateUser(ctx, u.ID, updateReq)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "Updated Name", u2.Name)

	// 3. Delete
	_, err = s.userSvc.DeleteUser(ctx, u.ID)
	require.NoError(s.T(), err)
	
	// 4. Verify Not Found on Login (since record is soft deleted logic or missing)
	loginReq := &models.UserLoginRequest{Username: "todelete", Password: "abc"}
	_, err = s.userSvc.Login(ctx, loginReq, nil)
	require.Error(s.T(), err)
}

func TestServiceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ServiceIntegrationTestSuite))
}
