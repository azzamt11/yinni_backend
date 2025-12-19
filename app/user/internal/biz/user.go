package biz

import (
	"context"
	"time"

	v1 "yinni_backend/api/helloworld/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")
)

// User is a User model.
type User struct {
	ID        int64
	Name      string
	Age       int
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepo is a Greater repo.
type UserRepo interface {
	Create(context.Context, *User) (*User, error)
	Update(context.Context, *User) (*User, error)
	Delete(context.Context, int64) (*User, error)
	GetUser(context.Context, int64) (*User, error)
	ListAllUser(context.Context) ([]*User, error)
}

// UserUsecase is a User usecase.
type UserUsecase struct {
	repo UserRepo
	log  *log.Helper
}

// NewUserUsecase new a User usecase.
func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{repo: repo, log: log.NewHelper(logger)}
}

// CreateUser creates a User, and returns the new User.
func (uc *UserUsecase) CreateUser(ctx context.Context, u *User) (*User, error) {
	log.Infof("CreateUser: %v", u.Email)
	return uc.repo.Create(ctx, u)
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, u *User) (*User, error) {
	log.Infof("UpdateUser: %v", u.Email)
	return uc.repo.Update(ctx, u)
}

func (uc *UserUsecase) DeleteUser(ctx context.Context, id int64) (*User, error) {
	log.Infof("DeleteUser: %v", id)
	return uc.repo.Delete(ctx, id)
}

func (uc *UserUsecase) GetUser(ctx context.Context, id int64) (*User, error) {
	log.Infof("GetUser: %v", id)
	return uc.repo.GetUser(ctx, id)
}

func (uc *UserUsecase) ListAllUser(ctx context.Context) ([]*User, error) {
	return uc.repo.ListAllUser(ctx)
}
