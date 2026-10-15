package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
	"github.com/google/uuid"
)

type UserRepository interface {
	CreateUser(ctx context.Context, param *db.CreateUserParams) (*db.User, error)
	GetAllUsers(ctx context.Context) ([]db.GetAllUsersRow, error)
	GetUserByUsername(ctx context.Context, username string) (*db.GetUserByUsernameRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*db.GetUserByIDRow, error)
	GetUserByIDs(ctx context.Context, id []uuid.UUID) ([]db.GetUserByIDsRow, error)
	UpdateUser(ctx context.Context, param *db.UpdateUserParams) (*db.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) (*db.User, error)
	ExistUsernameorEmail(ctx context.Context, username string, email string) (*db.ExistUsernameorEmailRow, error)
}

type userRepository struct {
	sqlcQueries *db.Queries
}

func NewUserRepository(sqlcQueries *db.Queries) UserRepository {
	return &userRepository{sqlcQueries: sqlcQueries}
}

func (u *userRepository) CreateUser(ctx context.Context, param *db.CreateUserParams) (*db.User, error) {
	if param == nil {
		return nil, apperrors.ErrInvalidQuery
	}

	res, err := u.sqlcQueries.CreateUser(ctx, *param)
	if err != nil {
		slog.ErrorContext(ctx, "user_repo: failed to create user", slog.String("error", err.Error()))
		return nil, fmt.Errorf("user_repo: failed to create user: %w", err)
	}

	return &res, nil
}

func (u *userRepository) GetAllUsers(ctx context.Context) ([]db.GetAllUsersRow, error) {
	rows, err := u.sqlcQueries.GetAllUsers(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}

		return nil, fmt.Errorf("user_repo: failed to get all users: %w", err)
	}

	return rows, nil
}

func (u *userRepository) GetUserByUsername(ctx context.Context, username string) (*db.GetUserByUsernameRow, error) {
	row, err := u.sqlcQueries.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}

		return nil, fmt.Errorf("user_repo: failed to get user by username: %w", err)
	}

	return &row, nil
}

func (u *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*db.GetUserByIDRow, error) {
	row, err := u.sqlcQueries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("user_repo: failed to get user by id: %w", err)
	}

	return &row, nil
}

func (u *userRepository) GetUserByIDs(ctx context.Context, id []uuid.UUID) ([]db.GetUserByIDsRow, error) {
	row, err := u.sqlcQueries.GetUserByIDs(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("user_repo: failed to get users by ids: %w", err)
	}

	return row, nil
}

func (u *userRepository) UpdateUser(ctx context.Context, param *db.UpdateUserParams) (*db.User, error) {
	res, err := u.sqlcQueries.UpdateUser(ctx, *param)
	if err != nil {
		return nil, fmt.Errorf("user_repo: failed to update user: %w", err)
	}

	return &res, nil
}

func (u *userRepository) DeleteUser(ctx context.Context, id uuid.UUID) (*db.User, error) {
	res, err := u.sqlcQueries.DeleteUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user_repo: failed to delete user: %w", err)
	}

	return &res, nil
}

func (u *userRepository) ExistUsernameorEmail(ctx context.Context, username string, email string) (*db.ExistUsernameorEmailRow, error) {
	res, err := u.sqlcQueries.ExistUsernameorEmail(ctx, db.ExistUsernameorEmailParams{Username: username, Email: email})
	if err != nil {
		return nil, fmt.Errorf("user_repo: failed to check duplicate username and email: %w", err)
	}

	return &res, nil
}
