package data

import (
	"context"
	"errors"
	"strings"

	"yinni_backend/app/auth/internal/biz"
	"yinni_backend/ent"
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
	r.log.Debug("Creating user with:",
		"email", u.Email,
		"name", u.Name,
		"has_password", u.Password != "",
	)

	// Check if ent client is nil
	if r.data.ent == nil {
		r.log.Error("Ent client is nil!")
		return nil, errors.New("database client not initialized")
	}

	// Step by step creation
	r.log.Debug("Creating builder...")
	builder := r.data.ent.User.Create()
	if builder == nil {
		r.log.Error("Builder is nil!")
		return nil, errors.New("failed to create builder")
	}

	r.log.Debug("Setting email...")
	builder = builder.SetEmail(u.Email)
	if builder == nil {
		r.log.Error("Builder became nil after SetEmail!")
		return nil, errors.New("failed to set email")
	}

	r.log.Debug("Setting password...")
	builder = builder.SetPassword(u.Password)
	if builder == nil {
		r.log.Error("Builder became nil after SetPassword!")
		return nil, errors.New("failed to set password")
	}

	r.log.Debug("Setting name...")
	builder = builder.SetName(u.Name)
	if builder == nil {
		r.log.Error("Builder became nil after SetName!")
		return nil, errors.New("failed to set name")
	}

	r.log.Debug("Builder created, about to save...")

	entUser, err := builder.Save(ctx)

	if err != nil {
		r.log.Errorf("Database error creating user: %v", err)
		r.log.Errorf("Error type: %T", err)

		// If error contains "table" or "doesn't exist", the migration might have failed
		if strings.Contains(strings.ToLower(err.Error()), "table") ||
			strings.Contains(strings.ToLower(err.Error()), "doesn't exist") ||
			strings.Contains(strings.ToLower(err.Error()), "unknown") {

			r.log.Error("Table might not exist. Migration likely failed.")
			return nil, biz.NewAuthError("database schema not ready. Please check migration logs.", biz.ErrInternal)
		}

		return nil, err
	}

	return &biz.User{
		ID:        int64(entUser.ID),
		Email:     entUser.Email,
		Password:  entUser.Password,
		Name:      entUser.Name,
		CreatedAt: entUser.CreatedAt,
		UpdatedAt: entUser.UpdatedAt,
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
		CreatedAt: entUser.CreatedAt,
		UpdatedAt: entUser.UpdatedAt,
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
		CreatedAt: entUser.CreatedAt,
		UpdatedAt: entUser.UpdatedAt,
	}, nil
}
