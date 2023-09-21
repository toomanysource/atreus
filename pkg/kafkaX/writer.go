package kafkaX

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/segmentio/kafka-go"
)

var ErrKafkaWriter = errors.New("kafka writer error")

func Update(writer *kafka.Writer, key, value string) error {
	err := writer.WriteMessages(context.TODO(),
		kafka.Message{
			Partition: 0,
			Key:       []byte(key),
			Value:     []byte(value),
		})
	if err != nil {
		return errors.Join(ErrKafkaWriter, err)
	}
	log.Infof("update message success, key: %v, value: %v", key, value)
	return nil
}
