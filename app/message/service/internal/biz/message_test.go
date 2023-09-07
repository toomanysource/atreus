package biz

import (
	"context"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"

	"github.com/toomanysource/atreus/middleware"
)

type MockMessageRepo struct{}

func (m *MockMessageRepo) GetMessageList(ctx context.Context, toUserId uint32, preMsgTime int64) ([]*Message, error) {
	return []*Message{
		{
			Id: 1,
		},
	}, nil
}

func (m *MockMessageRepo) PublishMessage(ctx context.Context, toUserId uint32, content string) error {
	return nil
}

func (m *MockMessageRepo) InitStoreMessageQueue() {}

var (
	ctx      = context.Background()
	mockRepo *MockMessageRepo
	useCase  *MessageUseCase
)

func TestMain(m *testing.M) {
	ctx = context.WithValue(ctx, middleware.UserIdKey("user_id"), uint32(1))
	useCase = NewMessageUseCase(mockRepo, log.DefaultLogger)
	m.Run()
	os.Exit(0)
}

func TestMessageUsecase_GetMessageList(t *testing.T) {
	msgs, err := useCase.GetMessageList(ctx, 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(msgs))
}

func TestMessageUsecase_PublishMessage(t *testing.T) {
	err := useCase.PublishMessage(ctx, 1, 1, "hahah")
	assert.Nil(t, err)
	err = useCase.PublishMessage(ctx, 1, 0, "hahah")
	assert.NotNil(t, err)
}
