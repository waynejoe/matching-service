package biz

import "github.com/google/wire"

// ProviderSet 是业务层 Wire provider 集合。
var ProviderSet = wire.NewSet(NewMetrics, NewMatchingUsecase)
