CREATE TABLE `post_views`
(
    `id`        BIGINT   NOT NULL AUTO_INCREMENT COMMENT '浏览记录ID',
    `post_id`   BIGINT   NOT NULL COMMENT '被浏览的笔记ID',
    `user_id`   BIGINT   NOT NULL COMMENT '浏览者ID',
    `viewed_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '浏览时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_post` (`user_id`, `post_id`),
    KEY `idx_post_id` (`post_id`),
    KEY `idx_viewed_at` (`viewed_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='笔记浏览记录表';