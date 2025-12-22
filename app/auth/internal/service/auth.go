package service

import (
	"context"

	pb "yinni_backend/api/auth/v1"
)

type AuthService struct {
	pb.UnimplementedAuthServer
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) SignUp(ctx context.Context, req *pb.SignUpRequest) (*pb.SignUpReply, error) {
	return &pb.SignUpReply{}, nil
}
func (s *AuthService) SignIn(ctx context.Context, req *pb.SignInRequest) (*pb.SignInReply, error) {
	return &pb.SignInReply{}, nil
}
