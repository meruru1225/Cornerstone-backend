CREATE TABLE `conversations`
(
    `id`                 BIGINT   NOT NULL AUTO_INCREMENT COMMENT '会话ID',
    `user_a_id`          BIGINT   NOT NULL COMMENT '参与者A ID',
    `user_b_id`          BIGINT   NOT NULL COMMENT '参与者B ID',
    `last_message_at`    DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '最后消息时间 (用于排序会话列表)',
    `unread_count_a`     INT      NOT NULL DEFAULT 0 COMMENT 'A的未读数 (对应 user_a_id)',
    `unread_count_b`     INT      NOT NULL DEFAULT 0 COMMENT 'B的未读数 (对应 user_b_id)',
    `created_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '会话创建时间',
    `updated_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后活动时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_users_pair` (`user_a_id`, `user_b_id`),
    KEY `idx_user_a` (`user_a_id`),
    KEY `idx_user_b` (`user_b_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='实时通讯会话元数据';