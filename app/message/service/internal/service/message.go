package service

import (
	pb "Atreus/api/message/service/v1"
	"Atreus/app/message/service/internal/biz"
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type MessageService struct {
	pb.UnimplementedMessageServiceServer
	mu  *biz.MessageUsecase
	log *log.Helper
}

func NewMessageService(mu *biz.MessageUsecase, logger log.Logger) *MessageService {
	return &MessageService{
		mu:  mu,
		log: log.NewHelper(log.With(logger, "model", "service/Message")),
	}
}

func (s *MessageService) GetMessageList(ctx context.Context, req *pb.MessageListRequest) (*pb.MessageListReply, error) {
	return &pb.MessageListReply{}, nil
}
func (s *MessageService) MessageAction(ctx context.Context, req *pb.MessageActionRequest) (*pb.MessageActionReply, error) {
	return &pb.MessageActionReply{}, nil
}
