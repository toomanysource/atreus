package biz

import (
	"errors"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewRelationUseCase)

const (
	FollowType   uint32 = 1
	UnfollowType uint32 = 2
)

var ErrInValidActionType = errors.New("invalid action type")
