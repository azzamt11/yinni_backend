package data

import (
	"context"

	"yinni_backend/app/prompt/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type promptRepo struct {
	data *Data
	log  *log.Helper
}

// NewPromptRepo .
func NewPromptRepo(data *Data, logger log.Logger) biz.PromptRepo {
	return &promptRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *promptRepo) Send(ctx context.Context, g *biz.Prompt) (*biz.Prompt, error) {
	return g, nil
}
