package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// Message is a Message model.
type Message struct {
}

// MessageRepo is a Greater repo.
type MessageRepo interface {
	// params: token,toUserId,msgTime
	GetMessageList(context.Context, string, int64, int64) ([]*Message, error)

	// params: token,toUserId,actionType,content
	PublishMessage(context.Context, string, int64, int32, string) error
}

// MessageUsecase is a Message usecase.
type MessageUsecase struct {
	repo MessageRepo
	log  *log.Helper
}

// NewMessageUsecase new a Message usecase.
func NewMessageUsecase(repo MessageRepo, logger log.Logger) *MessageUsecase {
	return &MessageUsecase{repo: repo, log: log.NewHelper(logger)}
}

// GetMessageList return a list of messages beginning on the given preMsgTime from RabbitMQ
func (uc *MessageUsecase) GetMessageList(ctx context.Context, token string, toUserId int64, preMsgTime int64) ([]*Message, error) {
	// firstly auth token

	// pull the message

	return nil, nil
}

// PublishMessage push a message
func (uc *MessageUsecase) PublishMessage(ctx context.Context, token string, toUSerId int64, actionType int32, content string) error {
	// firstly auto token

	// push the message

	return nil
}
