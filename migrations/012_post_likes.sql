CREATE TABLE `likes`
(
    `user_id`    BIGINT   NOT NULL COMMENT '用户ID',
    `post_id`    BIGINT   NOT NULL COMMENT '笔记ID',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '点赞时间',
    PRIMARY KEY (`user_id`, `post_id`),
    KEY `idx_post_id` (`post_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='点赞关系表 (M2M)';