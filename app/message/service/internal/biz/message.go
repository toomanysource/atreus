package biz

import (
	"context"

	"github.com/toomanysource/atreus/pkg/errorX"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	PublishMessage = 1
)

type Message struct {
	UId        uint64 `copier:"Id"`
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

type MessageUseCase struct {
	repo MessageRepo
	log  *log.Helper
}

func NewMessageUseCase(repo MessageRepo, logger log.Logger) *MessageUseCase {
	go repo.InitStoreMessageQueue()
	return &MessageUseCase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "model", "usecase/message")),
	}
}

func (uc *MessageUseCase) GetMessageList(
	ctx context.Context, toUserId uint32, preMsgTime int64,
) ([]*Message, error) {
	messages, err := uc.repo.GetMessageList(ctx, toUserId, preMsgTime)
	if err != nil {
		uc.log.Errorf("GetMessageList error: %v", err)
	}
	return messages, err
}

func (uc *MessageUseCase) PublishMessage(
	ctx context.Context, toUserId uint32, actionType uint32, content string,
) error {
	switch actionType {
	case PublishMessage:
		err := uc.repo.PublishMessage(ctx, toUserId, content)
		if err != nil {
			uc.log.Errorf("PublishMessage error: %v", err)
		}
		return err
	default:
		return errorX.ErrInValidActionType
	}
}
