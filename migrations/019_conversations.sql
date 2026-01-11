CREATE TABLE `conversations`
(
    `id`               BIGINT   NOT NULL AUTO_INCREMENT,
    `type`             TINYINT  NOT NULL DEFAULT 1 COMMENT '1-单聊, 2-群聊',
    `peer_key`         VARCHAR(64)       DEFAULT '' COMMENT '单聊时为uid1_uid2(从小到大), 群聊可存群ID',
    `max_msg_seq`      BIGINT   NOT NULL DEFAULT 0 COMMENT '当前会话最大序列号',
    `last_msg_type`    TINYINT           DEFAULT 1 COMMENT '最后一条消息类型',
    `last_msg_content` VARCHAR(255)      DEFAULT '' COMMENT '最后消息预览',
    `last_sender_id`   BIGINT            DEFAULT 0 COMMENT '最后发送者',
    `last_message_at`  DATETIME          DEFAULT CURRENT_TIMESTAMP,
    `created_at`       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_peer_key` (`peer_key`),
    KEY `idx_last_message_at` (`last_message_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;