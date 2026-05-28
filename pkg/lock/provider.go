package lock

import "github.com/google/wire"

// ProviderSet 是分布式锁 Wire provider 集合。
var ProviderSet = wire.NewSet(NewRedisLock)
