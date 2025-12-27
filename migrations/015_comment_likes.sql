CREATE TABLE `comment_likes`
(
    `user_id`    BIGINT   NOT NULL COMMENT '用户ID (逻辑关联 users.id)',
    `comment_id` BIGINT   NOT NULL COMMENT '评论ID (逻辑关联 post_comments.id)',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '点赞时间',
    PRIMARY KEY (`user_id`, `comment_id`),
    KEY `idx_comment_id` (`comment_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='评论点赞关系表 (M2M)';