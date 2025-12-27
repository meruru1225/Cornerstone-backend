CREATE TABLE `user_tags`
(
    `user_id` BIGINT NOT NULL COMMENT '用户ID (逻辑关联 users.id)',
    `tag_id`  BIGINT NOT NULL COMMENT '标签ID (逻辑关联 tags.id)',
    PRIMARY KEY (`user_id`, `tag_id`),
    KEY `idx_tag_id` (`tag_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户兴趣标签关系表 (M2M)';