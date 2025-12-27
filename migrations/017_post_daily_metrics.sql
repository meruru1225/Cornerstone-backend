CREATE TABLE `post_daily_metrics`
(
    `id`             BIGINT   NOT NULL AUTO_INCREMENT COMMENT '记录ID',
    `post_id`        BIGINT   NOT NULL COMMENT '笔记ID',
    `metric_date`    DATE     NOT NULL COMMENT '记录日期',
    `new_likes`      INT      NOT NULL DEFAULT 0 COMMENT '当日新增点赞数',
    `new_comments`   INT      NOT NULL DEFAULT 0 COMMENT '当日新增评论数',
    `new_collects`   INT      NOT NULL DEFAULT 0 COMMENT '当日新增收藏数',
    `new_views`      INT      NOT NULL DEFAULT 0 COMMENT '当日新增访问量',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_post_date` (`post_id`, `metric_date`),
    KEY `idx_metric_date` (`metric_date`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='笔记每日数据快照表 (分析模块)';