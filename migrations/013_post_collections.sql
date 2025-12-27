CREATE TABLE `collections`
(
    `user_id`    BIGINT   NOT NULL COMMENT '用户ID (逻辑关联 users.id)',
    `post_id`    BIGINT   NOT NULL COMMENT '笔记ID (逻辑关联 posts.id)',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '收藏时间',
    PRIMARY KEY (`user_id`, `post_id`),
    KEY `idx_post_id` (`post_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='收藏关系表 (M2M)';