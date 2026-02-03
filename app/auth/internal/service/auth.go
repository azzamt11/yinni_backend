package service

import (
	"context"
	"errors"

	pb "yinni_backend/api/auth/v1"
	"yinni_backend/app/auth/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/structpb"
)

type AuthService struct {
	pb.UnimplementedAuthServer
	uc     *biz.AuthUsecase
	logger *log.Helper
}

func NewAuthService(uc *biz.AuthUsecase, logger log.Logger) *AuthService {
	return &AuthService{
		uc:     uc,
		logger: log.NewHelper(log.With(logger, "module", "auth/service")),
	}
}

func (s *AuthService) SignUp(ctx context.Context, req *pb.SignUpRequest) (*pb.SignUpReply, error) {
	s.logger.Infow(
		"signup_request",
		"email", req.Email,
		"name", req.Name,
		"has_password", req.Password != "",
	)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		return nil, errors.New("email, password, and name are required")
	}

	user, _, err := s.uc.SignUp(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		return nil, errors.New("internal server error")
	}

	return &pb.SignUpReply{
		UserId: user.ID,
	}, nil
}

func (s *AuthService) SignIn(ctx context.Context, req *pb.SignInRequest) (*pb.SignInReply, error) {
	s.logger.Infow(
		"signin_request",
		"email", req.Email,
		"has_password", req.Password != "",
	)

	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}

	user, token, err := s.uc.SignIn(ctx, req.Email, req.Password)
	if err != nil {
		if authErr, ok := err.(*biz.AuthError); ok {
			switch authErr.Type {
			case biz.ErrInvalidCredentials:
				return nil, errors.New("invalid email or password")
			}
		}
		return nil, errors.New("internal server error")
	}

	// Convert user to protobuf Struct
	userStruct, err := structpb.NewStruct(map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
	if err != nil {
		s.logger.Errorw("failed_to_build_user_struct", "error", err)
		return nil, errors.New("internal server error")
	}

	return &pb.SignInReply{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.uc.JWTExpire().Seconds()),
		User:        userStruct,
	}, nil
}
