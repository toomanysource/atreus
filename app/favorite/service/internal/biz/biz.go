package biz

import (
	"errors"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewFavoriteUseCase)

const (
	Favorite   uint32 = 1
	UnFavorite uint32 = 2
)

var ErrInValidActionType = errors.New("invalid action type")
