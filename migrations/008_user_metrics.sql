CREATE TABLE `user_daily_metrics`
(
    `id`                BIGINT   NOT NULL AUTO_INCREMENT COMMENT '记录ID',
    `user_id`           BIGINT   NOT NULL COMMENT '用户ID',
    `metric_date`       DATE     NOT NULL COMMENT '记录日期',
    `total_followers`   INT      NOT NULL COMMENT '当日总粉丝数',
    `created_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_user_date` (`user_id`, `metric_date`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户每日数据快照表 (分析模块)';