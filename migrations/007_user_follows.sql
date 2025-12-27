CREATE TABLE `user_follows`
(
    `follower_id`  BIGINT   NOT NULL COMMENT '粉丝ID (主动关注者)',
    `following_id` BIGINT   NOT NULL COMMENT '被关注者ID',
    `created_at`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '关注时间',
    PRIMARY KEY (`follower_id`, `following_id`),
    KEY `idx_following_id` (`following_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户关注关系表 (M2M 自引用)';