package services

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
)

type userTracingDecorator struct {
	userService UserService
	tracer      trace.Tracer
}

func NewUserTracingDecorator(userService UserService) UserService {
	return &userTracingDecorator{
		userService: userService,
		tracer:      otel.Tracer("accounts-service"),
	}
}

func (d *userTracingDecorator) GetAllUsers(ctx context.Context) ([]entities.User, error) {
	ctx, span := d.tracer.Start(ctx, "UserService.GetAllUsers")
	defer span.End()

	users, err := d.userService.GetAllUsers(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return users, err
}

func (d *userTracingDecorator) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	ctx, span := d.tracer.Start(ctx, "UserService.GetUserByID")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", id.String()))

	user, err := d.userService.GetUserByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return user, err
}

func (d *userTracingDecorator) GetUserByIDs(ctx context.Context, IDs []uuid.UUID) ([]entities.User, error) {
	ctx, span := d.tracer.Start(ctx, "UserService.GetUserByIDs")
	defer span.End()

	users, err := d.userService.GetUserByIDs(ctx, IDs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return users, err
}

func (d *userTracingDecorator) UpdateUser(ctx context.Context, id uuid.UUID, req *models.UserUpdateRequest) (*entities.User, error) {
	ctx, span := d.tracer.Start(ctx, "UserService.UpdateUser")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", id.String()))

	user, err := d.userService.UpdateUser(ctx, id, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return user, err
}

func (d *userTracingDecorator) DeleteUser(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	ctx, span := d.tracer.Start(ctx, "UserService.DeleteUser")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", id.String()))

	user, err := d.userService.DeleteUser(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return user, err
}
