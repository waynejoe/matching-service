package conf

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Bootstrap 是服务启动配置。
type Bootstrap struct {
	Server Server `yaml:"server"` // Server 是服务监听配置
	Data   Data   `yaml:"data"`   // Data 是数据中间件配置
	Match  Match  `yaml:"match"`  // Match 是撮合业务配置
}

// Server 是对外服务配置。
type Server struct {
	GRPC    GRPC    `yaml:"grpc"`    // GRPC 是 gRPC 服务配置
	Metrics Metrics `yaml:"metrics"` // Metrics 是 Prometheus 监控配置
}

// GRPC 是 gRPC 监听配置。
type GRPC struct {
	Addr string `yaml:"addr"` // Addr 是 gRPC 监听地址
}

// Metrics 是 Prometheus 监听配置。
type Metrics struct {
	Addr string `yaml:"addr"` // Addr 是 Prometheus 指标监听地址
}

// Data 是数据访问配置。
type Data struct {
	MySQL    MySQL    `yaml:"mysql"`    // MySQL 是撮合库配置
	Redis    Redis    `yaml:"redis"`    // Redis 是分布式锁和缓存配置
	RocketMQ RocketMQ `yaml:"rocketmq"` // RocketMQ 是订单消息配置
}

// MySQL 是 MySQL 连接配置。
type MySQL struct {
	Driver string `yaml:"driver"` // Driver 是数据库驱动
	Source string `yaml:"source"` // Source 是数据库连接串
}

// Redis 是 Redis 连接配置。
type Redis struct {
	Addr     string `yaml:"addr"`     // Addr 是 Redis 地址
	Password string `yaml:"password"` // Password 是 Redis 密码
	DB       int    `yaml:"db"`       // DB 是 Redis 库编号
}

// RocketMQ 是 RocketMQ 消费配置。
type RocketMQ struct {
	NameServers           []string `yaml:"name_servers"`             // NameServers 是 NameServer 地址列表
	DepositTopic          string   `yaml:"deposit_topic"`            // DepositTopic 是入金主队列
	WithdrawTopic         string   `yaml:"withdraw_topic"`           // WithdrawTopic 是出金主队列
	ConsumerGroup         string   `yaml:"consumer_group"`           // ConsumerGroup 是消费组
	MaxReconsumeTimes     int32    `yaml:"max_reconsume_times"`      // MaxReconsumeTimes 是最大重试次数
	SuspendQueueMillis    int      `yaml:"suspend_queue_millis"`     // SuspendQueueMillis 是顺序消费失败后的延迟重试毫秒数
	DeadLetterTopicRemark string   `yaml:"dead_letter_topic_remark"` // DeadLetterTopicRemark 是死信队列说明
}

// Match 是撮合参数配置。
type Match struct {
	MinCompleteRate    int64 `yaml:"min_complete_rate"`     // MinCompleteRate 是最低成交比例，3000 表示 30%
	DepositTTLMin      int   `yaml:"deposit_ttl_min"`       // DepositTTLMin 是入金默认过期分钟数
	BasketLimit        int   `yaml:"basket_limit"`          // BasketLimit 是启动时每个分片加载篮子数量
	ShardLockTTLSecond int   `yaml:"shard_lock_ttl_second"` // ShardLockTTLSecond 是 Redis 分片锁秒数
	ExpireIntervalSec  int   `yaml:"expire_interval_sec"`   // ExpireIntervalSec 是超时扫描间隔秒数
	ExpireBatchSize    int   `yaml:"expire_batch_size"`     // ExpireBatchSize 是每轮超时扫描数量
}

// Load 读取配置文件。
func Load(path string) (*Bootstrap, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out Bootstrap
	if err := yaml.Unmarshal(bs, &out); err != nil {
		return nil, err
	}
	fillDefault(&out)
	return &out, nil
}

// fillDefault 填充默认配置。
func fillDefault(out *Bootstrap) {
	if out.Server.GRPC.Addr == "" {
		out.Server.GRPC.Addr = ":9203"
	}
	if out.Server.Metrics.Addr == "" {
		out.Server.Metrics.Addr = ":8203"
	}
	if out.Data.MySQL.Driver == "" {
		out.Data.MySQL.Driver = "mysql"
	}
	if out.Data.RocketMQ.ConsumerGroup == "" {
		out.Data.RocketMQ.ConsumerGroup = "matching-service"
	}
	if out.Data.RocketMQ.MaxReconsumeTimes == 0 {
		out.Data.RocketMQ.MaxReconsumeTimes = 16
	}
	if out.Data.RocketMQ.SuspendQueueMillis == 0 {
		out.Data.RocketMQ.SuspendQueueMillis = 1000
	}
	if out.Match.MinCompleteRate == 0 {
		out.Match.MinCompleteRate = 3000
	}
	if out.Match.DepositTTLMin == 0 {
		out.Match.DepositTTLMin = 30
	}
	if out.Match.BasketLimit == 0 {
		out.Match.BasketLimit = 10000
	}
	if out.Match.ShardLockTTLSecond == 0 {
		out.Match.ShardLockTTLSecond = 30
	}
	if out.Match.ExpireIntervalSec == 0 {
		out.Match.ExpireIntervalSec = 5
	}
	if out.Match.ExpireBatchSize == 0 {
		out.Match.ExpireBatchSize = 500
	}
}
