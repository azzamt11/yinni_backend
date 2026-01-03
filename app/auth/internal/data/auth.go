package data

import (
	"context"

	"yinni_backend/app/auth/internal/biz"
	"yinni_backend/ent"
	"yinni_backend/ent/user"

	"github.com/go-kratos/kratos/v2/log"
)

type authRepo struct {
	data *Data
	log  *log.Helper
}

// NewAuthRepo creates a new Auth repository.
func NewAuthRepo(data *Data, logger log.Logger) biz.AuthRepo {
	return &authRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// CreateUser creates a new user in the database.
func (r *authRepo) CreateUser(ctx context.Context, u *biz.User) (*biz.User, error) {
	// Create user in database
	entUser, err := r.data.ent.User.
		Create().
		SetEmail(u.Email).
		SetPassword(u.Password).
		SetName(u.Name).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Convert ent.User to biz.User
	return &biz.User{
		ID:        int64(entUser.ID),
		Email:     entUser.Email,
		Password:  entUser.Password,
		Name:      entUser.Name,
		CreatedAt: entUser.CreateTime,
		UpdatedAt: entUser.UpdateTime,
	}, nil
}

// FindByEmail finds a user by email.
func (r *authRepo) FindByEmail(ctx context.Context, email string) (*biz.User, error) {
	entUser, err := r.data.ent.User.
		Query().
		Where(user.Email(email)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // User not found, return nil without error
		}
		return nil, err
	}

	// Convert ent.User to biz.User
	return &biz.User{
		ID:        int64(entUser.ID),
		Email:     entUser.Email,
		Password:  entUser.Password,
		Name:      entUser.Name,
		CreatedAt: entUser.CreateTime,
		UpdatedAt: entUser.UpdateTime,
	}, nil
}

// GetUserByID retrieves a user by ID.
func (r *authRepo) GetUserByID(ctx context.Context, id int64) (*biz.User, error) {
	entUser, err := r.data.ent.User.
		Get(ctx, int(id))
	if err != nil {
		return nil, err
	}

	// Convert ent.User to biz.User
	return &biz.User{
		ID:        int64(entUser.ID),
		Email:     entUser.Email,
		Password:  entUser.Password,
		Name:      entUser.Name,
		CreatedAt: entUser.CreateTime,
		UpdatedAt: entUser.UpdateTime,
	}, nil
}
