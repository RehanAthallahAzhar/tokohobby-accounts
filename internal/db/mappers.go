package db

import (
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
)

func (u *User) ToDomain() *entities.User {
	if u == nil {
		return nil
	}
	return &entities.User{
		ID:          u.ID,
		Name:        u.Name,
		Username:    u.Username,
		Email:       u.Email,
		Role:        u.Role,
		Address:     u.Address,
		PhoneNumber: u.PhoneNumber,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func (u *GetUserByIDRow) ToDomain() *entities.User {
	if u == nil {
		return nil
	}
	return &entities.User{
		ID:          u.ID,
		Name:        u.Name,
		Username:    u.Username,
		Email:       u.Email,
		Role:        u.Role,
		Address:     u.Address,
		PhoneNumber: u.PhoneNumber,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func (u *GetUserByUsernameRow) ToDomain() *entities.User {
	if u == nil {
		return nil
	}
	return &entities.User{
		ID:          u.ID,
		Name:        u.Name,
		Username:    u.Username,
		Email:       u.Email,
		Role:        u.Role,
		Address:     u.Address,
		PhoneNumber: u.PhoneNumber,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func (u GetAllUsersRow) ToDomain() *entities.User {
	return &entities.User{
		ID:          u.ID,
		Name:        u.Name,
		Username:    u.Username,
		Email:       u.Email,
		Role:        u.Role,
		Address:     u.Address,
		PhoneNumber: u.PhoneNumber,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func (u GetUserByIDsRow) ToDomain() *entities.User {
	return &entities.User{
		ID:          u.ID,
		Name:        u.Name,
		Username:    u.Username,
		Email:       u.Email,
		Role:        u.Role,
		Address:     u.Address,
		PhoneNumber: u.PhoneNumber,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func ToDomainUsers[T interface{ ToDomain() *entities.User }](rows []T) []entities.User {
	res := make([]entities.User, 0, len(rows))
	for _, row := range rows {
		if domain := row.ToDomain(); domain != nil {
			res = append(res, *domain)
		}
	}
	return res
}
