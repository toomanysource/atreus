package biz

import (
	"context"
	"errors"

	"github.com/toomanysource/atreus/app/message/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

type Message struct {
	UId        uint64
	ToUserId   uint32
	FromUserId uint32
	Content    string
	CreateTime int64
}

type MessageRepo interface {
	GetMessageList(context.Context, uint32, int64) ([]*Message, error)
	PublishMessage(context.Context, uint32, string) error
	InitStoreMessageQueue()
}

type MessageUsecase struct {
	repo MessageRepo
	conf *conf.JWT
	log  *log.Helper
}

func NewMessageUsecase(repo MessageRepo, conf *conf.JWT, logger log.Logger) *MessageUsecase {
	go repo.InitStoreMessageQueue()
	return &MessageUsecase{
		repo: repo, conf: conf,
		log: log.NewHelper(log.With(logger, "model", "usecase/message")),
	}
}

func (uc *MessageUsecase) GetMessageList(
	ctx context.Context, toUserId uint32, preMsgTime int64,
) ([]*Message, error) {
	return uc.repo.GetMessageList(ctx, toUserId, preMsgTime)
}

func (uc *MessageUsecase) PublishMessage(ctx context.Context, toUserId uint32, actionType uint32, content string) error {
	switch actionType {
	case 1:
		return uc.repo.PublishMessage(ctx, toUserId, content)
	default:
		return errors.New("the actionType value for the error is provided")
	}
}
