CREATE TABLE `user_content_metrics`
(
    `id`             BIGINT   NOT NULL AUTO_INCREMENT COMMENT '记录ID',
    `user_id`        BIGINT   NOT NULL COMMENT '创作者用户ID (逻辑关联 users.id)',
    `metric_date`    DATE     NOT NULL COMMENT '记录日期',
    `total_likes`    INT      NOT NULL DEFAULT 0 COMMENT '当日总点赞数',
    `total_collects` INT      NOT NULL DEFAULT 0 COMMENT '当日总收藏数',
    `total_comments` INT      NOT NULL DEFAULT 0 COMMENT '当日总评论数',
    `total_views`    INT      NOT NULL DEFAULT 0 COMMENT '当日总访问量',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_user_date` (`user_id`, `metric_date`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户内容表现聚合表 (创作者数据中心)';