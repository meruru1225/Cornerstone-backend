CREATE TABLE `roles`
(
    `id`          BIGINT       NOT NULL AUTO_INCREMENT COMMENT '角色ID',
    `name`        VARCHAR(50)  NOT NULL COMMENT '角色名称 (e.g., "USER", "ADMIN")',
    `description` VARCHAR(255) DEFAULT NULL COMMENT '角色描述 (e.g., "普通用户", "管理员")',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_role_name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='角色表';