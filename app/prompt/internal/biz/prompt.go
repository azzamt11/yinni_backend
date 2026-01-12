package biz

import (
	"context"
	"strconv"
)

type PromptType int

const (
	PromptUnknown PromptType = iota
	PromptFindItem
	PromptSelectOption
	PromptMakePayment
)

type PromptResult struct {
	Type            PromptType
	Data            map[string]interface{}
	ServiceResponse string
}

type PromptRepo interface {
	Classify(ctx context.Context, prompt string) (PromptType, string, error)

	FindItem(ctx context.Context, query string) (map[string]interface{}, error)
	SelectItem(ctx context.Context, id int64) (map[string]interface{}, error)
	MakePayment(ctx context.Context, method string) (map[string]interface{}, error)
}

type PromptUsecase struct {
	repo PromptRepo
}

func NewPromptUsecase(repo PromptRepo) *PromptUsecase {
	return &PromptUsecase{repo: repo}
}

func (uc *PromptUsecase) HandlePrompt(ctx context.Context, prompt string) (*PromptResult, error) {

	t, value, err := uc.repo.Classify(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}

	switch t {
	case PromptFindItem:
		data, err = uc.repo.FindItem(ctx, value)
	case PromptSelectOption:
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		data, err = uc.repo.SelectItem(ctx, id)
	case PromptMakePayment:
		data, err = uc.repo.MakePayment(ctx, value)
	default:
		return &PromptResult{
			Type:            PromptUnknown,
			Data:            map[string]interface{}{"error": "unknown prompt"},
			ServiceResponse: "Unsupported prompt",
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return &PromptResult{
		Type:            t,
		Data:            data,
		ServiceResponse: "OK",
	}, nil
}
