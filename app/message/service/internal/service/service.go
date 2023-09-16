package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewMessageService)

const (
	CodeSuccess = 0
	CodeFailed  = 300
)
