package kafkaX

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
)

// Reader 消费者循环
// reader 消费者队列
// log 日志
// f 消息处理函数
func Reader(reader *kafka.Reader, log *log.Helper,
	f func(ctx context.Context, reader *kafka.Reader, msg kafka.Message),
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 监听Ctrl+C退出信号
	signChan := make(chan os.Signal, 1)
	signal.Notify(signChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signChan
		cancel()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := reader.ReadMessage(ctx)
			if errors.Is(err, context.Canceled) {
				return
			}
			if err != nil {
				log.Errorf("read message error, err: %v", err)
				return
			}
			f(ctx, reader, msg)
			err = reader.CommitMessages(ctx, msg)
			if err != nil {
				log.Errorf("commit message error, err: %v", err)
				return
			}
			log.Infof("commit message success, %v-(%v)", msg.Key, msg.Value)
		}
	}
}
