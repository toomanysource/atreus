package kafkaX

import (
	"context"
	"errors"

	"github.com/toomanysource/atreus/pkg/errorX"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/segmentio/kafka-go"
)

func Update(writer *kafka.Writer, key, value string) error {
	err := writer.WriteMessages(context.TODO(),
		kafka.Message{
			Partition: 0,
			Key:       []byte(key),
			Value:     []byte(value),
		})
	if err != nil {
		return errors.Join(errorX.ErrKafkaWriter, err)
	}
	log.Infof("update message success, key: %v, value: %v", key, value)
	return nil
}
