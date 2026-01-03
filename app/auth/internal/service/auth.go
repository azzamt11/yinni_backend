package service

import (
	"context"
	"errors"

	pb "yinni_backend/api/auth/v1"
	"yinni_backend/app/auth/internal/biz"
)

type AuthService struct {
	pb.UnimplementedAuthServer
	uc *biz.AuthUsecase
}

func NewAuthService(uc *biz.AuthUsecase) *AuthService {
	return &AuthService{uc: uc}
}

func (s *AuthService) SignUp(ctx context.Context, req *pb.SignUpRequest) (*pb.SignUpReply, error) {
	// Validate request
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return nil, errors.New("email, password, and name are required")
	}

	// Call usecase
	user, _, err := s.uc.SignUp(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		// Handle specific auth errors
		if authErr, ok := err.(*biz.AuthError); ok {
			switch authErr.Type {
			case biz.ErrUserAlreadyExists:
				return nil, errors.New("user already exists")
			case biz.ErrInvalidCredentials:
				return nil, errors.New("invalid credentials")
			default:
				return nil, errors.New("internal server error")
			}
		}
		return nil, err
	}

	return &pb.SignUpReply{
		UserId: user.ID,
	}, nil
}

func (s *AuthService) SignIn(ctx context.Context, req *pb.SignInRequest) (*pb.SignInReply, error) {
	// Validate request
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}

	// Call usecase
	_, token, err := s.uc.SignIn(ctx, req.Email, req.Password)
	if err != nil {
		// Handle specific auth errors
		if authErr, ok := err.(*biz.AuthError); ok {
			switch authErr.Type {
			case biz.ErrInvalidCredentials:
				return nil, errors.New("invalid email or password")
			default:
				return nil, errors.New("internal server error")
			}
		}
		return nil, err
	}

	return &pb.SignInReply{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.uc.JWTExpire().Seconds()), // Assuming you add this method to AuthUsecase
	}, nil
}
