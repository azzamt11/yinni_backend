package service

import (
	"context"

	v1 "yinni_backend/api/main/v1"
	"yinni_backend/app/main/internal/biz"
)

// MainService is a greeter service.
type MainService struct {
	v1.UnimplementedMainServer

	uc *biz.MainUsecase
}

// NewMainService new a greeter service.
func NewMainService(uc *biz.MainUsecase) *MainService {
	return &MainService{uc: uc}
}

// SayHello implements helloworld.MainServer.
func (s *MainService) SendPrompt(ctx context.Context, in *v1.SendPromptRequest) (*v1.SendPromptReply, error) {
	g, err := s.uc.Prompt(ctx, &biz.Main{Prompt: in.Prompt, Response: in.Prompt})
	if err != nil {
		return nil, err
	}
	return &v1.SendPromptReply{Type: v1.PromptType_FIND_ITEM, Data: g.Response}, nil
}
