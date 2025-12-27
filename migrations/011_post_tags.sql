CREATE TABLE `post_tags`
(
    `post_id` BIGINT NOT NULL COMMENT '笔记ID (逻辑关联 posts.id)',
    `tag_id`  BIGINT NOT NULL COMMENT '标签ID (逻辑关联 tags.id)',
    PRIMARY KEY (`post_id`, `tag_id`),
    KEY `idx_tag_id` (`tag_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='笔记标签关系表 (M2M)';