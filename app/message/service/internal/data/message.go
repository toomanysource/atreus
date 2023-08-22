package data

import (
	"context"

	"Atreus/app/message/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type MessageRepo struct {
	data *Data
	log  *log.Helper
}

// NewMessageRepo .
func NewMessageRepo(data *Data, logger log.Logger) biz.MessageRepo {
	return &MessageRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *MessageRepo) GetMessageList(ctx context.Context, token string, toUSerId int64, preMsgTime int64) ([]*biz.Message, error) {
	// consume the message from RabbitMQ

	return nil, nil
}

func (r *MessageRepo) PublishMessage(ctx context.Context, token string, toUSerId int64, actionType int32, content string) error {
	// publish the message into RabbitMQ

	return nil
}
