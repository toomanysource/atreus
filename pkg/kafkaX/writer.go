package kafkaX

import (
	"context"
	"strconv"

	"github.com/segmentio/kafka-go"
)

func Update(writer *kafka.Writer, id uint32, num int32) error {
	return writer.WriteMessages(context.TODO(),
		kafka.Message{
			Key:   []byte(strconv.Itoa(int(id))),
			Value: []byte(strconv.Itoa(int(num))),
		})
}
