package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/helpers"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services"

	accountpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/account"
)

type AccountServer struct {
	accountpb.UnimplementedAccountServiceServer
	UserService services.UserService
}

func NewAccountServer(userService services.UserService) *AccountServer {
	return &AccountServer{UserService: userService}
}

func (s *AccountServer) GetUser(ctx context.Context, req *accountpb.GetUserRequest) (*accountpb.User, error) {
	userID := req.GetId()
	if userID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user ID cannot be empty")
	}

	uuid, err := helpers.StringToUUID(userID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID format")
	}

	user, err := s.UserService.GetUserByID(ctx, uuid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return &accountpb.User{
		Id:          user.ID.String(),
		Name:        user.Name,
		Username:    user.Username,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		Address:     user.Address,
	}, nil
}

func (s *AccountServer) GetUsers(ctx context.Context, req *accountpb.GetUsersRequest) (*accountpb.GetUsersResponse, error) {
	users, err := s.UserService.GetAllUsers(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get users: %v", err)
	}

	pbUsers := make([]*accountpb.User, 0, len(users))

	for _, user := range users {
		pbUsers = append(pbUsers, &accountpb.User{
			Id:          user.ID.String(),
			Name:        user.Name,
			Username:    user.Username,
			Email:       user.Email,
			PhoneNumber: user.PhoneNumber,
			Address:     user.Address,
			Role:        user.Role,
		})
	}

	return &accountpb.GetUsersResponse{
		Users: pbUsers,
	}, nil
}
