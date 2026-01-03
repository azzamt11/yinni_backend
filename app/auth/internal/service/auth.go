package service

import (
	"context"
	"errors"

	pb "yinni_backend/api/auth/v1"
	"yinni_backend/app/auth/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
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

	// Validate request
	if req.Email == "" || req.Password == "" || req.Name == "" {
		s.logger.Errorw("signup_validation_failed",
			"email_empty", req.Email == "",
			"password_empty", req.Password == "",
			"name_empty", req.Name == "",
		)
		return nil, errors.New("email, password, and name are required")
	}

	s.logger.Debug("calling usecase.SignUp")
	// Call usecase
	user, token, err := s.uc.SignUp(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		s.logger.Errorw("signup_usecase_failed",
			"error", err,
			"error_type", getErrorType(err),
		)

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

	s.logger.Infow("signup_success",
		"user_id", user.ID,
		"email", user.Email,
		"has_token", token != "",
		"token_length", len(token),
	)

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

	// Validate request
	if req.Email == "" || req.Password == "" {
		s.logger.Errorw("signin_validation_failed",
			"email_empty", req.Email == "",
			"password_empty", req.Password == "",
		)
		return nil, errors.New("email and password are required")
	}

	s.logger.Debug("calling usecase.SignIn")
	// Call usecase
	user, token, err := s.uc.SignIn(ctx, req.Email, req.Password)
	if err != nil {
		s.logger.Errorw("signin_usecase_failed",
			"error", err,
			"error_type", getErrorType(err),
			"email", req.Email,
		)

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

	s.logger.Infow("signin_success",
		"user_id", user.ID,
		"email", user.Email,
		"has_token", token != "",
		"token_length", len(token),
		"token_preview", getTokenPreview(token),
	)

	return &pb.SignInReply{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.uc.JWTExpire().Seconds()),
	}, nil
}

// Helper function to get error type
func getErrorType(err error) string {
	if err == nil {
		return "nil"
	}

	switch err.(type) {
	case *biz.AuthError:
		return "AuthError"
	default:
		return "Unknown"
	}
}

// Helper function to preview token (first 20 chars)
func getTokenPreview(token string) string {
	if len(token) > 20 {
		return token[:20] + "..."
	}
	return token
}
