package services

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging/rabbitmq"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"
	"github.com/RehanAthallahAzhar/tokohobby-messaging/kafka"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req *models.UserRegisterRequest) (*entities.User, error)
	Login(ctx context.Context, req *models.UserLoginRequest, metadata *models.ActivityMetadata) (*models.UserAuthResult, error)
	Logout(ctx context.Context, authHeader string) error
	RefreshSession(ctx context.Context, oldRefreshToken string) (*models.GenerateTokenResult, error)
}

type AuthServiceImpl struct {
	userRepo         repositories.UserRepository
	validator        *validator.Validate
	tokenService     token.TokenService
	JWTBlacklistRepo repositories.JWTBlacklistRepository
	RefreshTokenRepo repositories.RefreshTokenRepository
	eventPublisher   *rabbitmq.EventPublisher
	kafkaProducer    *kafka.ActivityProducer
	log              *slog.Logger
}

func NewAuthService(
	userRepo repositories.UserRepository,
	validator *validator.Validate,
	tokenService token.TokenService,
	jwtBlacklistRepo repositories.JWTBlacklistRepository,
	refreshTokenRepo repositories.RefreshTokenRepository,
	eventPublisher *rabbitmq.EventPublisher,
	kafkaProducer *kafka.ActivityProducer,
	log *slog.Logger,
) AuthService {
	return &AuthServiceImpl{
		userRepo:         userRepo,
		validator:        validator,
		tokenService:     tokenService,
		JWTBlacklistRepo: jwtBlacklistRepo,
		RefreshTokenRepo: refreshTokenRepo,
		eventPublisher:   eventPublisher,
		kafkaProducer:    kafkaProducer,
		log:              log,
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, req *models.UserRegisterRequest) (*entities.User, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, err
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
		s.log.ErrorContext(ctx, "failed to generate password hash",
			slog.String("username", req.Username),
			slog.Any("error", err),
		)

		return nil, err
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
		return nil, err
	}

	// safe async
	s.safeGo(ctx, func(asyncCtx context.Context) {
		userIDStr := userDB.ID.String()

		// Publish RabbitMQ
		if s.eventPublisher != nil {
			event := rabbitmq.UserRegisteredEvent{
				UserID:    userIDStr,
				Email:     userDB.Email,
				Username:  userDB.Username,
				CreatedAt: time.Now(),
			}
			if err := s.eventPublisher.PublishUserRegistered(asyncCtx, event); err != nil {
				s.log.ErrorContext(asyncCtx, "failed to publish user registered event",
					slog.String("user_id", userIDStr),
					slog.Any("error", err),
				)
			}
		}

		s.publishActivity(ctx, "REGISTER", userDB.ID.String(), "registration", map[string]interface{}{
			"username": userDB.Username,
			"email":    userDB.Email,
			"role":     userDB.Role,
		})
	})

	return userDB.ToDomain(), nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, req *models.UserLoginRequest, metadata *models.ActivityMetadata) (*models.UserAuthResult, error) {
	userDB, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(userDB.Password), []byte(req.Password))
	if err != nil {
		s.log.WarnContext(ctx, "invalid login attempt: password comparison failed",
			slog.String("username", req.Username),
			slog.String("user_id", userDB.ID.String()),
		)

		return nil, apperrors.ErrInvalidCredentials
	}

	accTokenReq := &models.GenerateAccessTokenRequest{
		UserID:   userDB.ID,
		Username: userDB.Username,
		Role:     userDB.Role,
	}

	// access token (1 hour)
	accessToken, err := s.tokenService.GenerateAccessToken(ctx, accTokenReq)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to generate access token",
			slog.String("user_id", userDB.ID.String()),
			slog.Any("error", err),
		)

		return nil, apperrors.ErrFailedToGenerateToken
	}

	// refresh token (7 days)
	refreshToken, err := s.tokenService.GenerateRefreshToken(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to generate refresh token",
			slog.String("user_id", userDB.ID.String()),
			slog.Any("error", err),
		)

		return nil, apperrors.ErrFailedToGenerateToken
	}

	// Store Refresh Token in Redis (7 days)
	err = s.RefreshTokenRepo.StoreRefreshToken(ctx, userDB.ID.String(), refreshToken, 7*24*time.Hour)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to store refresh token in redis",
			slog.String("user_id", userDB.ID.String()),
			slog.Any("error", err),
		)

		return nil, apperrors.ErrFailedToGenerateToken
	}

	user := userDB.ToDomain()

	res := &models.UserAuthResult{
		User: user,
		Token: &models.GenerateTokenResult{
			AccessToken:           accessToken,
			AccessTokenExpiresAt:  time.Now().Add(1 * time.Hour),
			RefreshToken:          refreshToken,
			RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		},
	}

	s.publishActivity(ctx, "LOGIN", user.ID.String(), metadata.SessionID, map[string]interface{}{
		"ip_address": metadata.IPAddress,
		"user_agent": metadata.UserAgent,
		"username":   user.Username,
	})

	return res, nil
}

func (s *AuthServiceImpl) Logout(ctx context.Context, authHeader string) error {
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		s.log.WarnContext(ctx, "invalid logout attempt: token string format is invalid",
			slog.String("auth_header", authHeader),
		)
		return apperrors.ErrInvalidTokenFormat
	}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &models.JWTClaims{})
	if err != nil {
		s.log.WarnContext(ctx, "invalid logout attempt: token parsing failed",
			slog.String("auth_header", authHeader),
			slog.Any("error", err),
		)
		return apperrors.ErrInvalidToken
	}

	claims, ok := token.Claims.(*models.JWTClaims)
	if !ok {
		s.log.WarnContext(ctx, "invalid logout attempt: token claims type is invalid",
			slog.String("auth_header", authHeader),
		)
		return apperrors.ErrInvalidToken
	}

	jti := claims.ID
	if jti == "" {
		s.log.WarnContext(ctx, "invalid logout attempt: missing JTI in token claims",
			slog.String("auth_header", authHeader),
		)
		return apperrors.ErrMissingJTI
	}

	remainingTime := time.Until(claims.ExpiresAt.Time)
	if remainingTime < 0 {
		s.log.WarnContext(ctx, "invalid logout attempt: token has expired",
			slog.String("auth_header", authHeader),
		)
		return apperrors.ErrExpiredToken
	}

	err = s.JWTBlacklistRepo.AddToBlacklist(ctx, jti, remainingTime)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to add token to blacklist",
			slog.String("jti", jti),
			slog.Any("error", err),
		)
		return apperrors.ErrFailedToRevokeToken
	}

	s.publishActivity(ctx, "LOGOUT", claims.UserID.String(), jti, map[string]interface{}{
		"username": claims.Username,
	})

	return nil
}

func (s *AuthServiceImpl) RefreshSession(ctx context.Context, oldRefreshToken string) (*models.GenerateTokenResult, error) {
	span := trace.SpanFromContext(ctx)

	userIDStr, err := s.RefreshTokenRepo.ValidateRefreshToken(ctx, oldRefreshToken)
	if err != nil || userIDStr == "" {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid or expired refresh token")

		slog.ErrorContext(ctx, "Failed to validate refresh token", slog.Any("error", err))

		return nil, apperrors.ErrInvalidToken
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid user ID format in session")

		return nil, apperrors.ErrSessionError
	}

	userSvc, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "user not found")

		s.log.ErrorContext(ctx, "Failed to retrieve user by id", slog.Any("error", err))

		return nil, apperrors.ErrUserNotFound
	}

	req := &models.GenerateAccessTokenRequest{
		UserID:   userSvc.ID,
		Username: userSvc.Username,
		Role:     userSvc.Role,
	}

	// Generate NEW Access Token
	newAccessToken, err := s.tokenService.GenerateAccessToken(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate access token during refresh")

		s.log.ErrorContext(ctx, "Failed to generate access token", slog.Any("error", err))

		return nil, apperrors.ErrFailedToGenerateToken
	}

	// Generate NEW Refresh Token
	newRefreshToken, err := s.tokenService.GenerateRefreshToken(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate new refresh token during rotation")

		s.log.ErrorContext(ctx, "Failed to generate new refresh token", slog.Any("error", err))

		return nil, apperrors.ErrFailedToGenerateToken
	}

	// Revoke old Refresh Token
	if err := s.RefreshTokenRepo.RevokeRefreshToken(ctx, oldRefreshToken); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to revoke old refresh token")

		s.log.ErrorContext(ctx, "Failed to revoke old refresh token", slog.Any("error", err))
	}

	// Store new Refresh Token
	if err := s.RefreshTokenRepo.StoreRefreshToken(ctx, userIDStr, newRefreshToken, 7*24*time.Hour); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to store new refresh token")

		s.log.ErrorContext(ctx, "Failed to store new refresh token", slog.Any("error", err))

		return nil, apperrors.ErrFailedToGenerateToken
	}

	span.SetAttributes(
		attribute.String("user.id", userSvc.ID.String()),
		attribute.String("user.username", userSvc.Username),
	)

	return &models.GenerateTokenResult{
		AccessToken:           newAccessToken,
		AccessTokenExpiresAt:  time.Now().Add(1 * time.Hour),
		RefreshToken:          newRefreshToken,
		RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}, nil
}

func (s *AuthServiceImpl) safeGo(parentCtx context.Context, fn func(asyncCtx context.Context)) {
	// release the connection of HTTP Request but give a timeout (e.g., 10 seconds)
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), 10*time.Second)

	go func() {
		defer cancel()

		// prevent server crash if there is panic
		defer func() {
			if r := recover(); r != nil {
				s.log.ErrorContext(bgCtx, "recovered from panic in background goroutine",
					slog.Any("panic", r),
				)
			}
		}()

		fn(bgCtx)
	}()
}

func (s *AuthServiceImpl) publishActivity(ctx context.Context, eventType string, userID string, sessionID string, metadata map[string]any) {
	if s.kafkaProducer == nil {
		return
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	if sessionID == "" {
		sessionID = "no-session"
	}

	s.safeGo(ctx, func(asyncCtx context.Context) {
		if traceID := extractTraceID(asyncCtx); traceID != "" {
			metadata["trace_id"] = traceID
		}

		event := &kafka.UserActivityEvent{
			EventType: eventType,
			UserID:    &userID,
			SessionID: sessionID,
			Metadata:  metadata,
		}

		if err := s.kafkaProducer.PublishActivity(asyncCtx, event); err != nil {
			s.log.ErrorContext(asyncCtx, "failed to publish activity event to Kafka",
				slog.String("event_type", eventType),
				slog.String("user_id", userID),
				slog.Any("error", err),
			)
		}
	})
}
