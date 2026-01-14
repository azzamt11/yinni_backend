package service

import (
	"context"

	pb "yinni_backend/api/main/v1"
	"yinni_backend/app/main/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// PromptService implements pb.PromptServer
type MainService struct {
	pb.UnimplementedMainServer
	uc  *biz.PromptUsecase
	log *log.Helper
}

// NewPromptService creates a new PromptService
func NewMainService(uc *biz.PromptUsecase, logger log.Logger) *MainService {
	return &MainService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *MainService) SendPrompt(ctx context.Context, req *pb.SendPromptRequest) (*pb.SendPromptReply, error) {
	s.log.Info("SendPrompt called with prompt:", req.Prompt)

	result, err := s.uc.HandlePrompt(ctx, req.Prompt)
	if err != nil {
		s.log.Error("HandlePrompt error:", err)
		return nil, err
	}

	s.log.Info("SendPrompt completed, type:", result.Type)

	data, err := structpb.NewStruct(result.Data)
	if err != nil {
		s.log.Errorf("invalid struct data: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid response data")
	}

	return &pb.SendPromptReply{
		Type:            toProtoType(result.Type),
		Data:            data.String(),
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
