package biz

import (
	"errors"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewMessageUseCase)

const (
	PublishMessage = 1
)

var ErrInValidActionType = errors.New("invalid action type")
