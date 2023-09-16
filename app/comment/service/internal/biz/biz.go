package biz

import (
	"errors"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewCommentUseCase)

const (
	CreateType uint32 = 1
	DeleteType uint32 = 2
)

var (
	ErrCommentTextEmpty  = errors.New("comment text is empty")
	ErrInValidActionType = errors.New("invalid action type")
	ErrInvalidId         = errors.New("invalid id")
)
