CREATE TABLE `conversations`
(
    `id`                BIGINT   NOT NULL AUTO_INCREMENT COMMENT '会话ID',
    `type`              TINYINT  NOT NULL DEFAULT 1 COMMENT '会话类型: 1-单聊, 2-群聊',
    `max_msg_seq`       BIGINT   NOT NULL DEFAULT 0 COMMENT '当前消息的最大序列号 (单调递增)',
    `last_msg_content`  VARCHAR(255)      DEFAULT '' COMMENT '最后一条消息预览',
    `last_sender_id`    BIGINT            DEFAULT 0 COMMENT '最后一条消息的发送者',
    `last_message_at`   DATETIME          DEFAULT CURRENT_TIMESTAMP COMMENT '最后消息时间 (排序用)',
    `created_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_last_message_at` (`last_message_at`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT ='IM会话主表';

CREATE TABLE `conversation_members`
(
    `id`              BIGINT NOT NULL AUTO_INCREMENT,
    `conversation_id` BIGINT NOT NULL COMMENT '关联会话ID',
    `user_id`         BIGINT NOT NULL COMMENT '用户ID',
    `read_msg_seq`    BIGINT NOT NULL DEFAULT 0 COMMENT '用户在该会话中已读的最大序列号',
    `is_muted`        TINYINT NOT NULL DEFAULT 0 COMMENT '是否免打扰: 0-否, 1-是',
    `joined_at`       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_conv_user` (`conversation_id`, `user_id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT ='IM会话成员关系表';