package data

import (
	"github.com/google/wire"
)

// ProviderSet 是数据层 Wire provider 集合。
var ProviderSet = wire.NewSet(NewData)
