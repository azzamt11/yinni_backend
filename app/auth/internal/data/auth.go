package data

import (
	"context"
	"strings"

	"yinni_backend/app/auth/internal/biz"
	"yinni_backend/ent"
	"yinni_backend/ent/migrate"
	"yinni_backend/ent/user"

	"github.com/go-kratos/kratos/v2/log"
)

type authRepo struct {
	data *Data
	log  *log.Helper
}

// NewAuthRepo .
func NewAuthRepo(data *Data, logger log.Logger) biz.AuthRepo {
	return &authRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *authRepo) CreateUser(ctx context.Context, u *biz.User) (*biz.User, error) {
	// Try to create the user
	entUser, err := r.data.ent.User.
		Create().
		SetEmail(u.Email).
		SetPassword(u.Password).
		SetName(u.Name).
		Save(ctx)

	if err != nil {
		// Check if the error is because the users table doesn't exist
		if strings.Contains(err.Error(), "doesn't exist") ||
			strings.Contains(err.Error(), "table") ||
			strings.Contains(err.Error(), "unknown table") {

			r.log.Warn("Users table doesn't exist, creating it...")

			// Create the users table using Ent's migration
			if err := r.data.ent.Schema.Create(ctx, migrate.WithDropIndex(false), migrate.WithDropColumn(false)); err != nil {
				r.log.Errorf("Failed to create users table: %v", err)
				return nil, err
			}

			r.log.Info("Users table created successfully")

			// Try creating the user again
			return r.CreateUser(ctx, u)
		}
		return nil, err
	}

	return &biz.User{
		ID:        int64(entUser.ID),
		Email:     entUser.Email,
		Password:  entUser.Password,
		Name:      entUser.Name,
		CreatedAt: entUser.CreateTime,
		UpdatedAt: entUser.UpdateTime,
	}, nil
}

func (r *authRepo) FindByEmail(ctx context.Context, email string) (*biz.User, error) {
	entUser, err := r.data.ent.User.
		Query().
		Where(user.Email(email)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // User not found, return nil without error
		}

		// Check if table doesn't exist
		if strings.Contains(err.Error(), "doesn't exist") ||
			strings.Contains(err.Error(), "table") ||
			strings.Contains(err.Error(), "unknown table") {
			return nil, nil // Table doesn't exist, so user doesn't exist
		}

		return nil, err
	}

	return &biz.User{
		ID:        int64(entUser.ID),
		Email:     entUser.Email,
		Password:  entUser.Password,
		Name:      entUser.Name,
		CreatedAt: entUser.CreateTime,
		UpdatedAt: entUser.UpdateTime,
	}, nil
}

func (r *authRepo) GetUserByID(ctx context.Context, id int64) (*biz.User, error) {
	entUser, err := r.data.ent.User.
		Get(ctx, int(id))
	if err != nil {
		return nil, err
	}

	return &biz.User{
		ID:        int64(entUser.ID),
		Email:     entUser.Email,
		Password:  entUser.Password,
		Name:      entUser.Name,
		CreatedAt: entUser.CreateTime,
		UpdatedAt: entUser.UpdateTime,
	}, nil
}
