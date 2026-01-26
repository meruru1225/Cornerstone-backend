CREATE TABLE `users`
(
    `id`              BIGINT   NOT NULL AUTO_INCREMENT COMMENT '用户唯一ID',
    `username`        VARCHAR(50)       DEFAULT NULL COMMENT '登录用户名',
    `phone`           VARCHAR(30)       DEFAULT NULL COMMENT '手机号',
    `password`        VARCHAR(255)      DEFAULT NULL COMMENT '密码',
    `is_ban`          TINYINT(1)        DEFAULT 0 COMMENT '是否封禁',
    `is_delete`       TINYINT(1)        DEFAULT 0 COMMENT '是否注销',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`),
    UNIQUE KEY `idx_phone` (`phone`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 
  AUTO_INCREMENT=10001 COMMENT ='用户表';

INSERT INTO `users` (`id`, `username`, `password`)
VALUES (10000, 'admin', '$2a$10$7czhmrnCjk/0w7DdAIrufumMdqlhH3vXoofiO.QosufEk4KkLGnqy');
