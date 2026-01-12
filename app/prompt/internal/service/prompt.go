package service

import (
	"context"

	pb "yinni_backend/api/prompt/v1"
	"yinni_backend/app/prompt/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/structpb"
)

// PromptService implements pb.PromptServer
type PromptService struct {
	pb.UnimplementedPromptServer
	uc  *biz.PromptUsecase
	log *log.Helper
}

// NewPromptService creates a new PromptService
func NewPromptService(uc *biz.PromptUsecase, logger log.Logger) *PromptService {
	return &PromptService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *PromptService) SendPrompt(ctx context.Context, req *pb.SendPromptRequest) (*pb.SendPromptReply, error) {

	result, err := s.uc.HandlePrompt(ctx, req.Prompt)
	if err != nil {
		return nil, err
	}

	data, _ := structpb.NewStruct(result.Data)

	return &pb.SendPromptReply{
		Type:            toProtoType(result.Type),
		Data:            data,
		ServiceResponse: result.ServiceResponse,
	}, nil
}

func toProtoType(t biz.PromptType) pb.PromptType {
	switch t {
	case biz.PromptFindItem:
		return pb.PromptType_FIND_ITEM
	case biz.PromptSelectOption:
		return pb.PromptType_SELECT_OPTION
	case biz.PromptMakePayment:
		return pb.PromptType_MAKE_PAYMENT
	default:
		return pb.PromptType_UNKNOWN
	}
}
