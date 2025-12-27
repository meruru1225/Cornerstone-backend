CREATE TABLE `posts`
(
    `id`             BIGINT       NOT NULL AUTO_INCREMENT COMMENT '笔记ID',
    `user_id`        BIGINT       NOT NULL COMMENT '作者ID',
    `title`          VARCHAR(255)          DEFAULT NULL COMMENT '笔记标题',
    `content`        TEXT         NOT NULL COMMENT '笔记正文',
    `likes_count`    INT          NOT NULL DEFAULT 0 COMMENT '点赞数',
    `comments_count` INT          NOT NULL DEFAULT 0 COMMENT '评论数',
    `collects_count` INT          NOT NULL DEFAULT 0 COMMENT '收藏数',
    `status`         TINYINT      NOT NULL DEFAULT 0 COMMENT '状态(0:审核中, 1:已发布, 2:拒绝, 3:待人工)',
    `is_deleted`     TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '逻辑删除标志 (0:否, 1:是)',
    `created_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='笔记主表';