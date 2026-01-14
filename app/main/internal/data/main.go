package data

import (
	"context"

	"yinni_backend/app/main/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type mainRepo struct {
	data *Data
	log  *log.Helper
}

// NewMainRepo .
func NewMainRepo(data *Data, logger log.Logger) biz.MainRepo {
	return &mainRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *mainRepo) Prompt(ctx context.Context, g *biz.Main) (*biz.Main, error) {
	return &biz.Main{
		Prompt:   g.Prompt,
		Response: g.Response,
	}, nil
}
