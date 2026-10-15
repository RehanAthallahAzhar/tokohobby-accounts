package services

import (
	"context"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type authTracingDecorator struct {
	authService AuthService
	tracer      trace.Tracer
}

func NewAuthTracingDecorator(authService AuthService) AuthService {
	return &authTracingDecorator{
		authService: authService,
		tracer:      otel.Tracer("auth-service"),
	}
}

func (d *authTracingDecorator) Register(ctx context.Context, req *models.UserRegisterRequest) (*entities.User, error) {
	ctx, span := d.tracer.Start(ctx, "authService.Register")
	defer span.End()

	span.SetAttributes(attribute.String("username", req.Username), attribute.String("email", req.Email))

	user, err := d.authService.Register(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return user, err
}

func (d *authTracingDecorator) Login(ctx context.Context, req *models.UserLoginRequest, metadata *models.ActivityMetadata) (*models.UserAuthResult, error) {
	ctx, span := d.tracer.Start(ctx, "authService.Login")
	defer span.End()

	span.SetAttributes(attribute.String("username", req.Username))

	user, err := d.authService.Login(ctx, req, metadata)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return user, err
}

func (d *authTracingDecorator) Logout(ctx context.Context, authHeader string) error {
	ctx, span := d.tracer.Start(ctx, "UserService.Logout")
	defer span.End()

	err := d.authService.Logout(ctx, authHeader)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (d *authTracingDecorator) RefreshSession(ctx context.Context, oldRefreshToken string) (*models.GenerateTokenResult, error) {
	ctx, span := d.tracer.Start(ctx, "UserService.RefreshSession")
	defer span.End()

	user, err := d.authService.RefreshSession(ctx, oldRefreshToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return user, err
}
