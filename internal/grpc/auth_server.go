package grpc

import (
	"context"

	authpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	TokenService token.TokenService
}

func NewAuthServer(tokenService token.TokenService) *AuthServer {
	return &AuthServer{TokenService: tokenService}
}

func (s *AuthServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	tokenString := req.GetToken()

	isValid, userID, username, role, errMsg, err := s.TokenService.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal server error during token validation: %v", err)
	}

	if !isValid {
		return &authpb.ValidateTokenResponse{
			IsValid:      false,
			ErrorMessage: errMsg,
		}, status.Errorf(codes.Unauthenticated, "Token validation failed: %s", errMsg)
	}

	return &authpb.ValidateTokenResponse{
		IsValid:      true,
		UserId:       userID.String(),
		Username:     username,
		Role:         role,
		ErrorMessage: "",
	}, nil
}
