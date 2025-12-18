package data

import (
	"context"

	"kratos_go_microservices_template/app/user/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type userRepo struct {
	data *Data
	log  *log.Helper
}

// NewUserRepo .
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *userRepo) Create(ctx context.Context, g *biz.User) (*biz.User, error) {
	return g, nil
}

func (r *userRepo) Update(ctx context.Context, g *biz.User) (*biz.User, error) {
	return g, nil
}

func (r *userRepo) GetUser(context.Context, int64) (*biz.User, error) {
	return nil, nil
}

func (r *userRepo) Delete(context.Context, int64) (*biz.User, error) {
	return nil, nil
}

func (r *userRepo) ListAllUser(context.Context) ([]*biz.User, error) {
	return nil, nil
}
