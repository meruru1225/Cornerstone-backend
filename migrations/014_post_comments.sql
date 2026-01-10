CREATE TABLE `post_comments`
(
    `id`               BIGINT        NOT NULL AUTO_INCREMENT COMMENT '评论ID',
    `post_id`          BIGINT        NOT NULL COMMENT '关联的笔记ID',
    `user_id`          BIGINT        NOT NULL COMMENT '评论者ID (谁发的)',
    `content`          VARCHAR(1000) NOT NULL COMMENT '评论内容',
    `media_info`       JSON                   DEFAULT NULL COMMENT '媒体列表JSON',
    `root_id`          BIGINT        NOT NULL DEFAULT 0 COMMENT '根评论ID (0:这是一级评论)',
    `parent_id`        BIGINT        NOT NULL DEFAULT 0 COMMENT '直接父评论ID (0:这是直接评论帖子)',
    `reply_to_user_id` BIGINT        NOT NULL DEFAULT 0 COMMENT '被回复的用户ID (0:无)',
    `likes_count`      INT           NOT NULL DEFAULT 0 COMMENT '点赞数',
    `status`           TINYINT(1)    NOT NULL DEFAULT 0 COMMENT '评论状态',
    `is_deleted`       TINYINT(1)    NOT NULL DEFAULT 0 COMMENT '逻辑删除',
    `created_at`       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_post_id` (`post_id`),
    KEY `idx_root_id` (`root_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='笔记评论表';