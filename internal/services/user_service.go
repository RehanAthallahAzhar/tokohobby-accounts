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
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/RehanAthallahAzhar/shopeezy-accounts/internal/db"
	apperrors "github.com/RehanAthallahAzhar/shopeezy-accounts/internal/pkg/errors"

	"github.com/RehanAthallahAzhar/shopeezy-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/shopeezy-accounts/internal/models"
	"github.com/RehanAthallahAzhar/shopeezy-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/shopeezy-accounts/internal/services/token"
)

type UserSource interface {
	db.GetAllUsersRow |
		db.GetUserByIDRow |
		db.GetUserByIDsRow |
		db.User
}

type UserService interface {
	Register(ctx context.Context, req *models.UserRegisterRequest) (*entities.User, error)
	Login(ctx context.Context, req *models.UserLoginRequest) (*entities.User, error)
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
	log              *logrus.Logger
}

func NewUserService(
	userRepo repositories.UserRepository,
	validator *validator.Validate,
	tokenService token.TokenService,
	JWTBlacklistRepo repositories.JWTBlacklistRepository,
	log *logrus.Logger,
) UserService {
	return &UserServiceImpl{
		userRepo:         userRepo,
		validator:        validator,
		tokenService:     tokenService,
		JWTBlacklistRepo: JWTBlacklistRepo,
		log:              log,
	}
}

func (s *UserServiceImpl) Register(ctx context.Context, req *models.UserRegisterRequest) (*entities.User, error) {
	var user *entities.User

	if err := s.validator.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)

		var errorMessages []string
		for _, fieldErr := range validationErrors {
			errorMessages = append(errorMessages, fmt.Sprintf("Field '%s' failed on the '%s' tag", fieldErr.Field(), fieldErr.Tag()))
		}

		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidRequestPayload, strings.Join(errorMessages, ", "))
	}

	if req.Role == "" {
		req.Role = "user"
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("service: Failed to register user: %w", err)
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

	user = &entities.User{
		ID:        userDB.ID,
		Name:      userDB.Name,
		Username:  userDB.Username,
		Email:     userDB.Email,
		Role:      userDB.Role,
		Address:   userDB.Address,
		CreatedAt: userDB.CreatedAt,
		UpdatedAt: userDB.UpdatedAt,
	}

	return user, nil
}

func (s *UserServiceImpl) Login(ctx context.Context, req *models.UserLoginRequest) (*entities.User, error) {
	var user *entities.User

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

	user = &entities.User{
		ID:          userDB.ID,
		Name:        userDB.Name,
		Username:    userDB.Username,
		Email:       userDB.Email,
		Role:        userDB.Role,
		Address:     userDB.Address,
		PhoneNumber: userDB.PhoneNumber,
		CreatedAt:   userDB.CreatedAt,
		UpdatedAt:   userDB.UpdatedAt,
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
