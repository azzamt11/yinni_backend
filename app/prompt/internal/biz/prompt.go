package biz

import (
	"context"

	v1 "yinni_backend/api/prompt/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrUserNotFound is user not found.
	ErrPromptInvalid = errors.NotFound(v1.ErrorReason_INVALID_PROMPT.String(), "invalid prompt")
)

// Prompt is a Prompt model.
type Prompt struct {
	Prompt string
}

// PromptRepo is a Greater repo.
type PromptRepo interface {
	Send(context.Context, *Prompt) (*Prompt, error)
}

// PromptUsecase is a Prompt usecase.
type PromptUsecase struct {
	repo PromptRepo
}

// NewPromptUsecase new a Prompt usecase.
func NewPromptUsecase(repo PromptRepo) *PromptUsecase {
	return &PromptUsecase{repo: repo}
}

// CreatePrompt creates a Prompt, and returns the new Prompt.
func (uc *PromptUsecase) SendPrompt(ctx context.Context, g *Prompt) (*Prompt, error) {
	log.Infof("SendPrompt: %v", g.Prompt)
	return uc.repo.Send(ctx, g)
}
