package service

import (
	"context"
	"encoding/json"

	pb "yinni_backend/api/prompt/v1"
	"yinni_backend/app/prompt/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	s.log.Info("SendPrompt called with prompt:", req.Prompt)

	result, err := s.uc.HandlePrompt(ctx, req.Prompt)
	if err != nil {
		s.log.Error("HandlePrompt error:", err)
		return nil, err
	}

	// --- FIX START ---
	// 1. Convert the result.Data map to JSON bytes
	jsonBytes, err := json.Marshal(result.Data)
	if err != nil {
		s.log.Errorf("failed to marshal result data: %v", err)
		return nil, status.Errorf(codes.Internal, "data serialization failed")
	}

	// 2. Unmarshal the JSON bytes into a standard map[string]interface{}
	var intermediateMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &intermediateMap); err != nil {
		s.log.Errorf("failed to unmarshal intermediate data: %v", err)
		return nil, status.Errorf(codes.Internal, "data conversion failed")
	}

	// 3. Now NewStruct will accept it because it's all standard types
	data, err := structpb.NewStruct(intermediateMap)
	// --- FIX END ---

	if err != nil {
		s.log.Errorf("invalid struct data: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid response data")
	}

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
