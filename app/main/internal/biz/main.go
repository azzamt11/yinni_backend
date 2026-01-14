package biz

import (
	"context"

	v1 "yinni_backend/api/main/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(v1.ErrorReason_INVALID_PROMPT.String(), "invalid prompt")
)

// Main is a Main model.
type Main struct {
	Prompt   string
	Response string
}

// MainRepo is a Greater repo.
type MainRepo interface {
	Prompt(context.Context, *Main) (*Main, error)
}

// MainUsecase is a Main usecase.
type MainUsecase struct {
	repo MainRepo
}

// NewMainUsecase new a Main usecase.
func NewMainUsecase(repo MainRepo) *MainUsecase {
	return &MainUsecase{repo: repo}
}

// CreateMain creates a Main, and returns the new Main.
func (uc *MainUsecase) Prompt(ctx context.Context, g *Main) (*Main, error) {
	log.Infof("Prompt: %v", g.Prompt)
	return uc.repo.Prompt(ctx, g)
}
