package repositories

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

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
	db  *db.Queries
	log *logrus.Logger
}

func NewUserRepository(sqlcQueries *db.Queries, log *logrus.Logger) UserRepository {
	return &userRepository{db: sqlcQueries, log: log}
}

func (u *userRepository) CreateUser(ctx context.Context, param *db.CreateUserParams) (*db.User, error) {
	var res db.User

	if param == nil {
		return nil, apperrors.ErrInvalidQuery
	}

	res, err := u.db.CreateUser(ctx, *param)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &res, nil
}

func (u *userRepository) GetAllUsers(ctx context.Context) ([]db.GetAllUsersRow, error) {
	u.log.Debug()
	var rows []db.GetAllUsersRow

	rows, err := u.db.GetAllUsers(ctx)
	if err != nil {
		u.log.WithError(err).Error("Failed to retrieve all users from the database")
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}

	return rows, nil
}

func (u *userRepository) GetUserByUsername(ctx context.Context, username string) (*db.GetUserByUsernameRow, error) {
	var row db.GetUserByUsernameRow
	row, err := u.db.GetUserByUsername(ctx, username)
	if err != nil {
		u.log.WithError(err).Error("Failed to retrieve user by username from the database")
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &row, nil
}

func (u *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*db.GetUserByIDRow, error) {
	var row db.GetUserByIDRow

	row, err := u.db.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &row, nil
}

func (u *userRepository) GetUserByIDs(ctx context.Context, id []uuid.UUID) ([]db.GetUserByIDsRow, error) {

	var row []db.GetUserByIDsRow

	row, err := u.db.GetUserByIDs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return row, nil
}

func (u *userRepository) UpdateUser(ctx context.Context, param *db.UpdateUserParams) (*db.User, error) {
	var res db.User

	res, err := u.db.UpdateUser(ctx, *param)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &res, nil
}

func (u *userRepository) DeleteUser(ctx context.Context, id uuid.UUID) (*db.User, error) {
	var res db.User

	res, err := u.db.DeleteUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	return &res, nil
}

func (u *userRepository) ExistUsernameorEmail(ctx context.Context, username string, email string) (*db.ExistUsernameorEmailRow, error) {
	res, err := u.db.ExistUsernameorEmail(ctx, db.ExistUsernameorEmailParams{Username: username, Email: email})
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate username and email: %w", err)
	}

	return &res, nil
}
