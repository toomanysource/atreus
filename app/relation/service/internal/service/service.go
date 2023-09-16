package service

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewRelationService)

const (
	CodeSuccess = 0
	CodeFailed  = 300
)
