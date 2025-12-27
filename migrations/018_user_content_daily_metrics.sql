CREATE TABLE `user_content_daily_metrics`
(
    `id`                 BIGINT   NOT NULL AUTO_INCREMENT COMMENT '记录ID',
    `user_id`            BIGINT   NOT NULL COMMENT '创作者用户ID (逻辑关联 users.id)',
    `metric_date`        DATE     NOT NULL COMMENT '记录日期',
    `total_new_likes`    INT      NOT NULL DEFAULT 0 COMMENT '当日总新增点赞数',
    `total_new_comments` INT      NOT NULL DEFAULT 0 COMMENT '当日总新增评论数',
    `total_new_views`    INT      NOT NULL DEFAULT 0 COMMENT '当日总新增访问量',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_user_date` (`user_id`, `metric_date`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户内容表现聚合表 (创作者数据中心)';