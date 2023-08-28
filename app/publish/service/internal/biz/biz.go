package biz

import "github.com/google/wire"

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewPublishUsecase)
