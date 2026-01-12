package service

import (
	"context"

	pb "yinni_backend/api/prompt/v1"
	"yinni_backend/app/prompt/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type PromptService struct {
	pb.UnimplementedPromptServer
	uc  *biz.PromptUsecase
	log *log.Helper
}

func NewPromptService(uc *biz.PromptUsecase, logger log.Logger) *PromptService {
	return &PromptService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *PromptService) SendPrompt(ctx context.Context, req *pb.SendPromptRequest) (*pb.SendPromptReply, error) {
	return &pb.SendPromptReply{}, nil
}
