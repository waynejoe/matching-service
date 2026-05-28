CREATE DATABASE IF NOT EXISTS syntra_match
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;

USE syntra_match;

CREATE TABLE IF NOT EXISTS deposit_order (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  deposit_no VARCHAR(64) NOT NULL COMMENT '入金单号',
  merchant_id BIGINT NOT NULL DEFAULT 0 COMMENT '商户ID',
  channel VARCHAR(32) NOT NULL COMMENT '支付渠道',
  currency VARCHAR(16) NOT NULL COMMENT '币种',
  amount BIGINT NOT NULL COMMENT '入金金额，最小货币单位',
  status TINYINT NOT NULL COMMENT '状态：1待撮合 2已锁定 3已撮合 4失败 5过期',
  expire_at DATETIME(3) NOT NULL COMMENT '过期时间',
  matched_basket_no VARCHAR(64) DEFAULT NULL COMMENT '匹配到的出金篮子单号',
  match_no VARCHAR(64) DEFAULT NULL COMMENT '撮合结果单号',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_deposit_no (deposit_no),
  KEY idx_match_pool (channel, currency, status, expire_at),
  KEY idx_amount (channel, currency, amount),
  KEY idx_match_no (match_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='入金单';

CREATE TABLE IF NOT EXISTS withdraw_basket (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  basket_no VARCHAR(64) NOT NULL COMMENT '篮子单号',
  withdraw_no VARCHAR(64) NOT NULL COMMENT '出金单号',
  merchant_id BIGINT NOT NULL DEFAULT 0 COMMENT '商户ID',
  channel VARCHAR(32) NOT NULL COMMENT '支付渠道',
  currency VARCHAR(16) NOT NULL COMMENT '币种',
  target_amount BIGINT NOT NULL COMMENT '出金目标金额',
  current_amount BIGINT NOT NULL DEFAULT 0 COMMENT '已凑金额',
  status TINYINT NOT NULL COMMENT '状态：1等待 2锁定 3已撮合 4过期 5取消',
  expire_at DATETIME(3) NOT NULL COMMENT '出金过期时间',
  version BIGINT NOT NULL DEFAULT 0 COMMENT '乐观锁版本',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_basket_no (basket_no),
  UNIQUE KEY uk_withdraw_no (withdraw_no),
  KEY idx_pool (channel, currency, status, expire_at),
  KEY idx_need_expr (channel, currency, status, target_amount, current_amount),
  KEY idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='出金撮合篮子';

CREATE TABLE IF NOT EXISTS basket_deposit (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  basket_no VARCHAR(64) NOT NULL COMMENT '篮子单号',
  withdraw_no VARCHAR(64) NOT NULL COMMENT '出金单号',
  deposit_no VARCHAR(64) NOT NULL COMMENT '入金单号',
  amount BIGINT NOT NULL COMMENT '入金金额，最小货币单位',
  status TINYINT NOT NULL COMMENT '状态：1已挂入 2已撮合 3已释放',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_deposit_no (deposit_no),
  KEY idx_basket_no (basket_no),
  KEY idx_withdraw_no (withdraw_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='篮子入金明细';

CREATE TABLE IF NOT EXISTS match_record (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  match_no VARCHAR(64) NOT NULL COMMENT '撮合结果单号',
  basket_no VARCHAR(64) NOT NULL COMMENT '篮子单号',
  withdraw_no VARCHAR(64) NOT NULL COMMENT '出金单号',
  target_amount BIGINT NOT NULL COMMENT '出金目标金额',
  matched_amount BIGINT NOT NULL COMMENT '实际撮合金额',
  short_amount BIGINT NOT NULL DEFAULT 0 COMMENT '少发金额',
  channel VARCHAR(32) NOT NULL COMMENT '支付渠道',
  currency VARCHAR(16) NOT NULL COMMENT '币种',
  status TINYINT NOT NULL COMMENT '状态：1已创建 2已发送支付 3支付中 4成功 5失败',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_match_no (match_no),
  UNIQUE KEY uk_basket_no (basket_no),
  KEY idx_withdraw_no (withdraw_no),
  KEY idx_status (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='撮合结果';

CREATE TABLE IF NOT EXISTS match_record_deposit (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  match_no VARCHAR(64) NOT NULL COMMENT '撮合结果单号',
  withdraw_no VARCHAR(64) NOT NULL COMMENT '出金单号',
  deposit_no VARCHAR(64) NOT NULL COMMENT '入金单号',
  amount BIGINT NOT NULL COMMENT '入金金额，最小货币单位',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_deposit_no (deposit_no),
  KEY idx_match_no (match_no),
  KEY idx_withdraw_no (withdraw_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='撮合结果入金明细';

CREATE TABLE IF NOT EXISTS match_event_inbox (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  event_id VARCHAR(128) NOT NULL COMMENT '事件唯一ID',
  topic VARCHAR(128) NOT NULL COMMENT 'RocketMQ topic',
  biz_no VARCHAR(64) NOT NULL COMMENT '业务单号',
  status TINYINT NOT NULL COMMENT '状态：1处理中 2成功 3失败',
  error_msg VARCHAR(512) DEFAULT NULL COMMENT '错误信息',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_event_id (event_id),
  KEY idx_biz_no (biz_no),
  KEY idx_status (status, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='RocketMQ 消费幂等表';

CREATE TABLE IF NOT EXISTS match_state_log (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  biz_no VARCHAR(64) NOT NULL COMMENT '业务单号',
  biz_type VARCHAR(32) NOT NULL COMMENT '业务类型：deposit/basket/match/payment',
  from_status TINYINT DEFAULT NULL COMMENT '原状态',
  to_status TINYINT NOT NULL COMMENT '新状态',
  reason VARCHAR(128) DEFAULT NULL COMMENT '变更原因',
  operator VARCHAR(64) DEFAULT NULL COMMENT '操作来源',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (id),
  KEY idx_biz_no (biz_no),
  KEY idx_biz_type_status (biz_type, to_status, created_at),
  KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='撮合状态流水';
