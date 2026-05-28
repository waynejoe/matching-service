package service

import "github.com/google/wire"

// ProviderSet 是接口适配层 Wire provider 集合。
var ProviderSet = wire.NewSet(NewMatchingConsumer, NewMatchingService)
