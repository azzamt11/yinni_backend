package biz

import (
	"context"
	"log"
	"time"
	"yinni_backend/internal/conf"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// User is a User model.
type User struct {
	ID        int64
	Email     string
	Password  string // Hashed password
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// In the AuthUsecase struct in biz/auth.go
func (uc *AuthUsecase) JWTExpire() time.Duration {
	return uc.jwtExpire
}

// JWT Claims structure matching your middleware
type JWTClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthRepo is an Auth repository interface.
type AuthRepo interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
}

// AuthUsecase is an Auth usecase.
type AuthUsecase struct {
	repo      AuthRepo
	jwtSecret string
	jwtExpire time.Duration
}

func NewAuthUsecase(repo AuthRepo, c *conf.Auth) (*AuthUsecase, error) {
	// Convert seconds to proper time.Duration
	jwtExpireSeconds := c.JwtExpire
	if jwtExpireSeconds == 0 {
		jwtExpireSeconds = 86400 // default 24 hours in seconds
	}

	// IMPORTANT: Convert seconds to nanoseconds for time.Duration
	jwtExpire := time.Duration(jwtExpireSeconds) * time.Second

	// DEBUG logging
	log.Printf("JWT Expire Configuration: %d seconds = %v duration",
		jwtExpireSeconds, jwtExpire)

	return &AuthUsecase{
		repo:      repo,
		jwtSecret: c.JwtSecret,
		jwtExpire: jwtExpire,
	}, nil
}

// HashPassword generates bcrypt hash of the password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPassword compares password with hash
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateJWTToken creates a JWT token for the user
func (uc *AuthUsecase) generateJWTToken(userID int64) (string, error) {
	expirationTime := time.Now().Add(uc.jwtExpire)

	claims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.jwtSecret))
}

// SignUp creates a new user
func (uc *AuthUsecase) SignUp(ctx context.Context, email, password, name string) (*User, string, error) {
	// Check if user already exists
	existingUser, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", NewAuthError("failed to check existing user: "+err.Error(), ErrInternal)
	}

	if existingUser != nil {
		return nil, "", NewAuthError("user already exists", ErrUserAlreadyExists)
	}

	// Hash password
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, "", NewAuthError("failed to hash password: "+err.Error(), ErrInternal)
	}

	// Create user
	user := &User{
		Email:    email,
		Password: hashedPassword,
		Name:     name,
	}

	createdUser, err := uc.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, "", NewAuthError("failed to create user: "+err.Error(), ErrInternal)
	}

	// Generate JWT token
	token, err := uc.generateJWTToken(createdUser.ID)
	if err != nil {
		return nil, "", NewAuthError("failed to generate token: "+err.Error(), ErrInternal)
	}

	return createdUser, token, nil
}

// SignIn authenticates a user
func (uc *AuthUsecase) SignIn(ctx context.Context, email, password string) (*User, string, error) {
	// Find user by email
	user, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", NewAuthError("failed to find user: "+err.Error(), ErrInternal)
	}

	if user == nil {
		return nil, "", NewAuthError("invalid email or password", ErrInvalidCredentials)
	}

	// Check password
	if !checkPassword(password, user.Password) {
		return nil, "", NewAuthError("invalid email or password", ErrInvalidCredentials)
	}

	// Generate JWT token
	token, err := uc.generateJWTToken(user.ID)
	if err != nil {
		return nil, "", NewAuthError("failed to generate token: "+err.Error(), ErrInternal)
	}

	return user, token, nil
}

// GetUserByID retrieves a user by ID
func (uc *AuthUsecase) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return uc.repo.GetUserByID(ctx, id)
}

// Error handling
type AuthErrorType string

const (
	ErrInvalidCredentials AuthErrorType = "INVALID_CREDENTIALS"
	ErrUserAlreadyExists  AuthErrorType = "USER_ALREADY_EXISTS"
	ErrInternal           AuthErrorType = "INTERNAL_ERROR"
)

type AuthError struct {
	Message string
	Type    AuthErrorType
}

func (e *AuthError) Error() string {
	return e.Message
}

func NewAuthError(message string, errorType AuthErrorType) *AuthError {
	return &AuthError{
		Message: message,
		Type:    errorType,
	}
}
