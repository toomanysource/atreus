package service

import (
	"context"

	pb "github.com/toomanysource/atreus/api/message/service/v1"
	"github.com/toomanysource/atreus/app/message/service/internal/biz"

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
	message, err := s.mu.GetMessageList(ctx, req.ToUserId, req.PreMsgTime)
	if err != nil {
		return &pb.MessageListReply{
			StatusCode: -1,
			StatusMsg:  err.Error(),
		}, nil
	}
	ml := make([]*pb.Message, 0, len(message))
	for _, m := range message {
		ml = append(ml, &pb.Message{
			Id:         m.UId,
			ToUserId:   m.ToUserId,
			FromUserId: m.FromUserId,
			Content:    m.Content,
			CreateTime: m.CreateTime,
		})
	}
	return &pb.MessageListReply{
		StatusCode:  0,
		StatusMsg:   "success",
		MessageList: ml,
	}, nil
}

func (s *MessageService) MessageAction(ctx context.Context, req *pb.MessageActionRequest) (*pb.MessageActionReply, error) {
	reply := &pb.MessageActionReply{StatusCode: 0, StatusMsg: "success"}
	err := s.mu.PublishMessage(ctx, req.ToUserId, req.ActionType, req.Content)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}
