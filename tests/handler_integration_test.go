package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/handlers"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services"
)

// ============================================================
// Handler Integration Test Suite
// ============================================================

type HandlerIntegrationTestSuite struct {
	IntegrationTestSuite
	handler *handlers.UserHandler
	e       *echo.Echo
	mockTs  *MockTokenService
	mockBl  *MockJWTBlacklistRepo
	mockRt  *MockRefreshTokenRepo
}

func (s *HandlerIntegrationTestSuite) SetupSuite() {
	s.IntegrationTestSuite.SetupSuite()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	queries := db.New(s.DB)
	userRepo := repositories.NewUserRepository(queries, logger)

	s.mockBl = new(MockJWTBlacklistRepo)
	s.mockRt = new(MockRefreshTokenRepo)
	s.mockTs = new(MockTokenService)

	val := validator.New()
	userSvc := services.NewUserService(userRepo, val, s.mockTs, s.mockBl, nil, nil, logger)

	s.e = echo.New()
	s.handler = handlers.NewHandler(userRepo, userSvc, s.mockTs, s.mockBl, s.mockRt, logger)
}

// ── Helper: membuat user via API dan mengembalikan UUID-nya ──────────────────

func (s *HandlerIntegrationTestSuite) createUserViaAPI(username, email, password string) uuid.UUID {
	payload := &models.UserRegisterRequest{
		Name:     "Handler User",
		Username: username,
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.RegisterUser(c)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	id, _ := uuid.Parse(data["id"].(string))
	return id
}

// ────────────────────────────────────────────────────────────────────────────
// 1. POST /register — Happy Path & Sad Path
// ────────────────────────────────────────────────────────────────────────────

func (s *HandlerIntegrationTestSuite) TestHandler_Register_Happy() {
	payload := &models.UserRegisterRequest{
		Name:     "Happy Register",
		Username: "happyreg",
		Email:    "happyreg@test.com",
		Password: "Password123!",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.RegisterUser(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, rec.Code)

	// Verifikasi data benar-benar masuk DB
	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username=$1", "happyreg").Scan(&count)
	assert.Equal(s.T(), 1, count)
}

func (s *HandlerIntegrationTestSuite) TestHandler_Register_MissingField_ReturnsBadRequest() {
	// Kirim payload tanpa password (field wajib)
	payload := map[string]string{"username": "missingpw", "email": "miss@test.com"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.RegisterUser(c)
	require.NoError(s.T(), err) // handler menelan error & merespons JSON
	assert.GreaterOrEqual(s.T(), rec.Code, 400)
}

// ────────────────────────────────────────────────────────────────────────────
// 2. POST /login — Happy Path & Invalid Credentials
// ────────────────────────────────────────────────────────────────────────────

func (s *HandlerIntegrationTestSuite) TestHandler_Login_Happy() {
	s.createUserViaAPI("loginok", "loginok@test.com", "Pass123!")

	payload := &models.UserLoginRequest{Username: "loginok", Password: "Pass123!"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("User-Agent", "IntegrationTest/1.0")
	req.Header.Set("X-Session-Id", "sess-123")
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.Login(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), "access_token_integration_mock", data["token"])
	assert.Equal(s.T(), "refresh_token_integration_mock", data["refresh_token"])
}

func (s *HandlerIntegrationTestSuite) TestHandler_Login_WrongPassword_ReturnsUnauthorized() {
	s.createUserViaAPI("wrgpwuser", "wrgpw@test.com", "Correct123!")

	payload := &models.UserLoginRequest{Username: "wrgpwuser", Password: "Incorrect!"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.Login(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusUnauthorized, rec.Code)
}

// ────────────────────────────────────────────────────────────────────────────
// 3. GET /users — GetAllUsers
// ────────────────────────────────────────────────────────────────────────────

func (s *HandlerIntegrationTestSuite) TestHandler_GetAllUsers() {
	s.createUserViaAPI("alluser1", "all1@test.com", "Pass1!")
	s.createUserViaAPI("alluser2", "all2@test.com", "Pass2!")

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.GetAllUsers(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	users, ok := resp["data"].([]interface{})
	require.True(s.T(), ok)
	assert.GreaterOrEqual(s.T(), len(users), 2)
}

// ────────────────────────────────────────────────────────────────────────────
// 4. GET /users/:id — GetUserByID (requires userID in Echo context)
// ────────────────────────────────────────────────────────────────────────────

func (s *HandlerIntegrationTestSuite) TestHandler_GetUserByID_Happy() {
	id := s.createUserViaAPI("getbyid", "getbyid@test.com", "Pass!")

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)
	c.Set("userID", id) // Simulasi middleware JWT inject

	err := s.handler.GetUserByID(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), id.String(), data["id"].(string))
}

func (s *HandlerIntegrationTestSuite) TestHandler_GetUserByID_NoContext_ReturnsUnauthorized() {
	req := httptest.NewRequest(http.MethodGet, "/users/some-id", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)
	// Sengaja tidak di-set userID di Echo context => simulate unauthenticated request

	err := s.handler.GetUserByID(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusUnauthorized, rec.Code)
}

// ────────────────────────────────────────────────────────────────────────────
// 5. PUT /users/:id — UpdateUser
// ────────────────────────────────────────────────────────────────────────────

func (s *HandlerIntegrationTestSuite) TestHandler_UpdateUser_Happy() {
	id := s.createUserViaAPI("updateme", "updateme@test.com", "Pass!")

	updatePayload := &models.UserUpdateRequest{
		Name:     "Updated Handler Name",
		Username: "updateme",
		Email:    "updateme@test.com",
		Password: "Pass!",
	}
	body, _ := json.Marshal(updatePayload)
	req := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)
	c.Set("userID", id)

	err := s.handler.UpdateUser(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(s.T(), "Updated Handler Name", data["name"])
}

// ────────────────────────────────────────────────────────────────────────────
// 6. DELETE /users/:id — DeleteUser
// ────────────────────────────────────────────────────────────────────────────

func (s *HandlerIntegrationTestSuite) TestHandler_DeleteUser_Happy() {
	id := s.createUserViaAPI("deleteme", "deleteme@test.com", "Pass!")

	req := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id.String())

	err := s.handler.DeleteUser(c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, rec.Code)
}

// ────────────────────────────────────────────────────────────────────────────
// Test Runner
// ────────────────────────────────────────────────────────────────────────────

func TestHandlerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(HandlerIntegrationTestSuite))
}
