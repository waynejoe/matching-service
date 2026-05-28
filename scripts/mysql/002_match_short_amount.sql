SET @has_short_amount := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'match_record'
    AND COLUMN_NAME = 'short_amount'
);

SET @sql := IF(
  @has_short_amount = 0,
  'ALTER TABLE match_record ADD COLUMN short_amount BIGINT NOT NULL DEFAULT 0 COMMENT ''少发金额'' AFTER matched_amount',
  'DO 0'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_overpay_amount := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'match_record'
    AND COLUMN_NAME = 'overpay_amount'
);

SET @sql := IF(
  @has_overpay_amount > 0,
  'UPDATE match_record SET short_amount = GREATEST(target_amount - matched_amount, 0)',
  'DO 0'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
  @has_overpay_amount > 0,
  'ALTER TABLE match_record DROP COLUMN overpay_amount',
  'DO 0'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_max_amount := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'withdraw_basket'
    AND COLUMN_NAME = 'max_amount'
);

SET @sql := IF(
  @has_max_amount > 0,
  'ALTER TABLE withdraw_basket DROP COLUMN max_amount',
  'DO 0'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
