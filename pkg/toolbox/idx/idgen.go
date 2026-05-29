package idx

import (
	"fmt"
	"sync/atomic"
	"time"
)

var seq uint64

// New 生成带前缀的业务单号：{prefix}{yyyyMMddHHmmss}{6位序号}。
func New(prefix string) string {
	n := atomic.AddUint64(&seq, 1)
	return fmt.Sprintf("%s%s%06d", prefix, time.Now().Format("20060102150405"), n%1000000)
}
