package data

import (
	"context"
	"database/sql"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"matching-service/internal/conf"
)

// Data 保存数据层共享资源。
type Data struct {
	DB *gorm.DB // DB 是 MySQL 数据库连接
}

// NewData 根据配置创建数据层实例。
func NewData(cfg *conf.Bootstrap) (*Data, error) {
	db, err := gorm.Open(mysql.Open(cfg.Data.MySQL.Source), &gorm.Config{
		TranslateError: true,
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		return nil, err
	}
	return &Data{DB: db}, nil
}

// Close 关闭数据库连接。
func (d *Data) Close() error {
	sqlDB, err := d.sqlDB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Transaction 执行数据库事务。
func (d *Data) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return d.DB.WithContext(ctx).Transaction(fn)
}

// Ping 检查 MySQL 连接是否可用。
func (d *Data) Ping(ctx context.Context) error {
	sqlDB, err := d.sqlDB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// sqlDB 返回底层 sql.DB。
func (d *Data) sqlDB() (*sql.DB, error) {
	return d.DB.DB()
}
