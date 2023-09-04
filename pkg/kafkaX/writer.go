package kafkaX

import (
	"context"
	"errors"
	"strconv"

	"github.com/toomanysource/atreus/pkg/errorX"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/segmentio/kafka-go"
)

func Update(writer *kafka.Writer, id uint32, num int32) error {
	err := writer.WriteMessages(context.TODO(),
		kafka.Message{
			Partition: 0,
			Key:       []byte(strconv.Itoa(int(id))),
			Value:     []byte(strconv.Itoa(int(num))),
		})
	if err != nil {
		return errors.Join(errorX.ErrKafkaWriter, err)
	}
	log.Infof("update message success, id: %v, num: %v", id, num)
	return nil
}
