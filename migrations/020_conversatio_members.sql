CREATE TABLE `conversation_members`
(
    `id`              BIGINT   NOT NULL AUTO_INCREMENT,
    `conversation_id` BIGINT   NOT NULL,
    `user_id`         BIGINT   NOT NULL,
    `read_msg_seq`    BIGINT   NOT NULL DEFAULT 0,
    `is_muted`        TINYINT  NOT NULL DEFAULT 0 COMMENT '免打扰',
    `is_pinned`       TINYINT  NOT NULL DEFAULT 0 COMMENT '是否置顶',
    `is_visible`      TINYINT  NOT NULL DEFAULT 1 COMMENT '会话是否在列表可见',
    `joined_at`       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_conv_user` (`conversation_id`, `user_id`),
    KEY `idx_user_visible_pinned` (`user_id`, `is_visible`, `is_pinned` DESC)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;