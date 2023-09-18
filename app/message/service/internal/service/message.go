package service

import (
	"context"

	"github.com/jinzhu/copier"

	pb "github.com/toomanysource/atreus/api/message/service/v1"
	"github.com/toomanysource/atreus/app/message/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type MessageService struct {
	pb.UnimplementedMessageServiceServer
	mu  *biz.MessageUseCase
	log *log.Helper
}

func NewMessageService(mu *biz.MessageUseCase, logger log.Logger) *MessageService {
	return &MessageService{
		mu:  mu,
		log: log.NewHelper(log.With(logger, "model", "service/message")),
	}
}

func (s *MessageService) GetMessageList(ctx context.Context, req *pb.MessageListRequest) (*pb.MessageListReply, error) {
	reply := &pb.MessageListReply{StatusCode: CodeSuccess, StatusMsg: "success", MessageList: make([]*pb.Message, 0)}
	message, err := s.mu.GetMessageList(ctx, req.ToUserId, req.PreMsgTime)
	if err != nil {
		reply.StatusCode = CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	ml := make([]*pb.Message, 0, len(message))
	err = copier.Copy(&ml, &message)
	if err != nil {
		reply.StatusCode = CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	reply.MessageList = ml
	return reply, nil
}

func (s *MessageService) MessageAction(ctx context.Context, req *pb.MessageActionRequest) (*pb.MessageActionReply, error) {
	reply := &pb.MessageActionReply{StatusCode: CodeSuccess, StatusMsg: "success"}
	err := s.mu.PublishMessage(ctx, req.ToUserId, req.ActionType, req.Content)
	if err != nil {
		reply.StatusCode = CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}
