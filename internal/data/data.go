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
func NewData(cfg *conf.Bootstrap) (*Data, func(), error) {
	db, err := gorm.Open(mysql.Open(cfg.Data.Mysql.Source), &gorm.Config{
		TranslateError: true,
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			LogLevel:                  gormLogLevel(cfg.Data.GormLogLevel),
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		return nil, nil, err
	}
	d := &Data{DB: db}
	cleanup := func() {
		sqlDB, err := d.sqlDB()
		if err != nil {
			return
		}
		_ = sqlDB.Close()
	}
	return d, cleanup, nil
}

// gormLogLevel 将配置字符串映射为 GORM 日志级别。
func gormLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "info":
		return logger.Info
	case "warn", "":
		return logger.Warn
	default:
		return logger.Warn
	}
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
