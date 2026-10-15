package services

import (
	"context"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging/rabbitmq"
	"github.com/RehanAthallahAzhar/tokohobby-messaging/kafka"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"
)

type UserService interface {
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
	RefreshTokenRepo repositories.RefreshTokenRepository
	eventPublisher   *rabbitmq.EventPublisher
	kafkaProducer    *kafka.ActivityProducer
	log              *slog.Logger
}

func NewUserService(
	userRepo repositories.UserRepository,
	validator *validator.Validate,
	tokenService token.TokenService,
	JWTBlacklistRepo repositories.JWTBlacklistRepository,
	refreshTokenRepo repositories.RefreshTokenRepository,
	eventPublisher *rabbitmq.EventPublisher,
	kafkaProducer *kafka.ActivityProducer,
	log *slog.Logger,
) *UserServiceImpl {
	return &UserServiceImpl{
		userRepo:         userRepo,
		validator:        validator,
		tokenService:     tokenService,
		JWTBlacklistRepo: JWTBlacklistRepo,
		RefreshTokenRepo: refreshTokenRepo,
		eventPublisher:   eventPublisher,
		kafkaProducer:    kafkaProducer,
		log:              log,
	}
}

func (s *UserServiceImpl) GetAllUsers(ctx context.Context) ([]entities.User, error) {
	users, err := s.userRepo.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	return db.ToDomainUsers(users), nil
}

func (s *UserServiceImpl) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user.ToDomain(), nil
}

func (s *UserServiceImpl) GetUserByIDs(ctx context.Context, IDs []uuid.UUID) ([]entities.User, error) {
	users, err := s.userRepo.GetUserByIDs(ctx, IDs)
	if err != nil {
		return nil, err
	}

	return db.ToDomainUsers(users), nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, id uuid.UUID, req *models.UserUpdateRequest) (*entities.User, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, err
	}

	dbParams := &db.UpdateUserParams{
		ID:          id,
		Name:        *req.Name,
		Address:     *req.Address,
		PhoneNumber: *req.PhoneNumber,
	}

	user, err := s.userRepo.UpdateUser(ctx, dbParams)
	if err != nil {
		return nil, err
	}

	return user.ToDomain(), nil
}

func (s *UserServiceImpl) DeleteUser(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := s.userRepo.DeleteUser(ctx, id)
	if err != nil {
		return nil, err
	}

	return user.ToDomain(), nil
}

func extractTraceID(ctx context.Context) string {
	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}
	return ""
}
