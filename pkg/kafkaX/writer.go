package kafkaX

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/toomanysource/atreus/pkg/errorX"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/segmentio/kafka-go"
)

func Update(writer *kafka.Writer, key, value any) error {
	keys, err := json.Marshal(key)
	if err != nil {
		return errors.Join(errorX.ErrJsonMarshal, err)
	}
	values, err := json.Marshal(value)
	if err != nil {
		return errors.Join(errorX.ErrJsonMarshal, err)
	}
	err = writer.WriteMessages(context.TODO(),
		kafka.Message{
			Partition: 0,
			Key:       keys,
			Value:     values,
		})
	if err != nil {
		return errors.Join(errorX.ErrKafkaWriter, err)
	}
	log.Infof("update message success, key: %v, value: %v", string(keys), string(values))
	return nil
}
