package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging/rabbitmq"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-messaging/kafka"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"
)

// ActivityMetadata contains HTTP request metadata for activity tracking
type ActivityMetadata struct {
	SessionID string
	IPAddress string
	UserAgent string
}

type UserSource interface {
	db.GetAllUsersRow |
		db.GetUserByIDRow |
		db.GetUserByIDsRow |
		db.User |
		db.GetUserByUsernameRow
}

type UserService interface {
	Register(ctx context.Context, req *models.UserRegisterRequest) (*entities.User, error)
	Login(ctx context.Context, req *models.UserLoginRequest, metadata *ActivityMetadata) (*entities.User, error)
	Logout(ctx context.Context, authHeader string) error
	GetAllUsers(ctx context.Context) ([]entities.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetUserByIDs(ctx context.Context, IDs []uuid.UUID) ([]entities.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req *models.UserUpdateRequest) (*entities.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) (*entities.User, error)
}

type UserServiceImpl struct {
	userRepo         repositories.UserRepository
	validator        *validator.Validate
	tokenService     token.TokenService
	JWTBlacklistRepo repositories.JWTBlacklistRepository
	eventPublisher   *rabbitmq.EventPublisher
	kafkaProducer    *kafka.ActivityProducer
	log              *logrus.Logger
}

func NewUserService(
	userRepo repositories.UserRepository,
	validator *validator.Validate,
	tokenService token.TokenService,
	JWTBlacklistRepo repositories.JWTBlacklistRepository,
	eventPublisher *rabbitmq.EventPublisher,
	kafkaProducer *kafka.ActivityProducer,
	log *logrus.Logger,
) UserService {
	return &UserServiceImpl{
		userRepo:         userRepo,
		validator:        validator,
		tokenService:     tokenService,
		JWTBlacklistRepo: JWTBlacklistRepo,
		eventPublisher:   eventPublisher,
		kafkaProducer:    kafkaProducer,
		log:              log,
	}
}

func (s *UserServiceImpl) Register(ctx context.Context, req *models.UserRegisterRequest) (*entities.User, error) {
	if err := s.validator.Struct(req); err != nil {
		fieldErrors := err.(validator.ValidationErrors)

		var errorMessages []string
		for _, fieldErr := range fieldErrors {
			errorMessages = append(errorMessages, fmt.Sprintf("Field '%s' failed on the '%s' tag", fieldErr.Field(), fieldErr.Tag()))
		}

		return nil, fmt.Errorf("validation failed: %s", strings.Join(errorMessages, ", "))
	}

	var validationErrors []errors.ValidationError

	// Check for duplicate username & email
	existingUser, err := s.userRepo.ExistUsernameorEmail(ctx, req.Username, req.Email)
	if err == nil && existingUser != nil {
		if existingUser.Username == req.Username {
			validationErrors = append(validationErrors, errors.ValidationError{
				Field:   "username",
				Message: "username already exists",
			})
		}
		if existingUser.Email == req.Email {
			validationErrors = append(validationErrors, errors.ValidationError{
				Field:   "email",
				Message: "email already exists",
			})
		}
	}

	// check role
	if req.Role == "" {
		req.Role = "user"
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password hash: %w", err)
	}

	if len(validationErrors) > 0 {
		return nil, apperrors.ValidationErrors{Errors: validationErrors}
	}

	dbParam := &db.CreateUserParams{
		ID:          uuid.New(),
		Name:        req.Name,
		Username:    req.Username,
		Email:       req.Email,
		Password:    string(hashedPassword),
		PhoneNumber: "",
		Address:     "",
		Role:        req.Role,
	}

	userDB, err := s.userRepo.CreateUser(ctx, dbParam)
	if err != nil {
		return nil, fmt.Errorf("service: failed to register user: %w", err)
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		event := rabbitmq.UserRegisteredEvent{
			UserID:    userDB.ID.String(),
			Email:     userDB.Email,
			Username:  userDB.Username,
			CreatedAt: time.Now(),
		}
		if err := s.eventPublisher.PublishUserRegistered(ctx, event); err != nil {
			log.Errorf("Failed to publish user registered event: %v", err)
			// Don't fail the registration if event publish fails
		}
	}()

	return toDomainUser(userDB), nil
}

func (s *UserServiceImpl) Login(ctx context.Context, req *models.UserLoginRequest, metadata *ActivityMetadata) (*entities.User, error) {
	userDB, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve user by username from the database")
		return nil, fmt.Errorf("service: failed to login: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(userDB.Password), []byte(req.Password))
	if err != nil {
		s.log.WithError(err).Error("Password comparison failed")
		return nil, apperrors.ErrInvalidCredentials
	}

	user := toDomainUser(userDB)

	// Track login activity to Kafka (async, non-blocking)
	if s.kafkaProducer != nil && metadata != nil {
		go func() {
			userIDStr := user.ID.String()
			sessionID := metadata.SessionID
			if sessionID == "" {
				sessionID = "no-session"
			}

			if err := s.kafkaProducer.PublishActivity(context.Background(), &kafka.UserActivityEvent{
				EventType: "LOGIN",
				UserID:    &userIDStr,
				SessionID: sessionID,
				Metadata: map[string]interface{}{
					"ip_address": metadata.IPAddress,
					"user_agent": metadata.UserAgent,
					"username":   user.Username,
				},
			}); err != nil {
				s.log.WithError(err).Error("Failed to publish login activity to Kafka")
			}
		}()
	}

	return user, nil
}

func (s *UserServiceImpl) Logout(ctx context.Context, authHeader string) error {

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return apperrors.ErrInvalidTokenFormat
	}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &models.JWTClaims{})
	if err != nil {
		return apperrors.ErrInvalidToken
	}

	claims, ok := token.Claims.(*models.JWTClaims)
	if !ok {
		return apperrors.ErrInvalidToken
	}

	jti := claims.ID
	if jti == "" {
		return apperrors.ErrMissingJTI
	}

	remainingTime := time.Until(claims.ExpiresAt.Time)
	if remainingTime < 0 {
		return apperrors.ErrExpiredToken
	}

	err = s.JWTBlacklistRepo.AddToBlacklist(ctx, jti, remainingTime)
	if err != nil {
		return apperrors.ErrFailedToRevokeToken
	}

	return nil
}

func (s *UserServiceImpl) GetAllUsers(ctx context.Context) ([]entities.User, error) {
	users, err := s.userRepo.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get all users: %w", err)
	}

	return toDomainUsers(users), nil
}

func (s *UserServiceImpl) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get user by id: %w", err)
	}

	return toDomainUser(user), nil
}

func (s *UserServiceImpl) GetUserByIDs(ctx context.Context, IDs []uuid.UUID) ([]entities.User, error) {
	users, err := s.userRepo.GetUserByIDs(ctx, IDs)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get user by id: %w", err)
	}

	return toDomainUsers(users), nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, id uuid.UUID, req *models.UserUpdateRequest) (*entities.User, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidRequestPayload, err)
	}

	dbParams := &db.UpdateUserParams{
		ID:          id,
		Name:        req.Name,
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		Address:     req.Address,
		PhoneNumber: req.PhoneNumber,
	}

	user, err := s.userRepo.UpdateUser(ctx, dbParams)
	if err != nil {
		return nil, fmt.Errorf("UpdateUser service error: %w", err)
	}

	return toDomainUser(user), nil
}

func (s *UserServiceImpl) DeleteUser(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := s.userRepo.DeleteUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("UpdateUser service error: %w", err)
	}

	return toDomainUser(user), nil
}

func toDomainUser[T UserSource](dbUser *T) *entities.User {
	v := reflect.ValueOf(dbUser)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	id := v.FieldByName("ID").Interface().(uuid.UUID)

	return &entities.User{
		ID:          id,
		Name:        v.FieldByName("Name").Interface().(string),
		Username:    v.FieldByName("Username").Interface().(string),
		Email:       v.FieldByName("Email").Interface().(string),
		Role:        v.FieldByName("Role").Interface().(string),
		Address:     v.FieldByName("Address").Interface().(string),
		PhoneNumber: v.FieldByName("PhoneNumber").Interface().(string),
		CreatedAt:   v.FieldByName("CreatedAt").Interface().(time.Time),
		UpdatedAt:   v.FieldByName("UpdatedAt").Interface().(time.Time),
	}
}

func toDomainUsers[T UserSource](dbUsers []T) []entities.User {
	users := make([]entities.User, 0, len(dbUsers))

	for _, dbUser := range dbUsers {
		users = append(users, *toDomainUser(&dbUser))
	}
	return users
}
