package data

import (
	"context"

	"yinni_backend/app/user/internal/biz"

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
	row, err := r.data.ent.User.
		Create().
		SetName(g.Name).
		SetAge(g.Age).
		SetEmail(g.Email).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	return &biz.User{
		ID:    int64(row.ID),
		Name:  row.Name,
		Age:   row.Age,
		Email: row.Email,
	}, nil
}

func (r *userRepo) Update(ctx context.Context, g *biz.User) (*biz.User, error) {
	row, err := r.data.ent.User.
		UpdateOneID(int(g.ID)).
		SetName(g.Name).
		SetAge(g.Age).
		SetEmail(g.Email).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	return &biz.User{
		ID:    int64(row.ID),
		Name:  row.Name,
		Age:   row.Age,
		Email: row.Email,
	}, nil
}

func (r *userRepo) GetUser(ctx context.Context, id int64) (*biz.User, error) {
	row, err := r.data.ent.User.Get(ctx, int(id))

	if err != nil {
		return nil, err
	}

	return &biz.User{
		ID:    int64(row.ID),
		Name:  row.Name,
		Age:   row.Age,
		Email: row.Email,
	}, nil
}

func (r *userRepo) Delete(ctx context.Context, id int64) (*biz.User, error) {
	err := r.data.ent.User.DeleteOneID(int(id)).Exec(ctx)
	return nil, err
}

func (r *userRepo) ListAllUser(ctx context.Context) ([]*biz.User, error) {
	rows, err := r.data.ent.User.Query().All(ctx)

	if err != nil {
		return nil, err
	}

	rv := make([]*biz.User, 0, len(rows))
	for _, row := range rows {
		rv = append(rv, &biz.User{
			ID:    int64(row.ID),
			Name:  row.Name,
			Age:   row.Age,
			Email: row.Email,
		})
	}
	return rv, nil
}
