CREATE TABLE `user_interest_tags`
(
    `user_id`    BIGINT   NOT NULL COMMENT '用户ID',
    `interests`  JSON     NOT NULL COMMENT '兴趣画像快照数据 ',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后同步时间',
    PRIMARY KEY (`user_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户兴趣画像持久化表 (Write-Back 快照存储)';