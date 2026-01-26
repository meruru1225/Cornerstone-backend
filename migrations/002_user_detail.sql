CREATE TABLE `user_detail`
(
    `user_id`    BIGINT      NOT NULL COMMENT '用户ID',
    `nickname`   VARCHAR(50) NOT NULL COMMENT '用户昵称',
    `avatar_url` VARCHAR(512) DEFAULT 'default_avatar.png' COMMENT '头像',
    `bio`        VARCHAR(255) DEFAULT '' COMMENT '简介',
    `gender`     TINYINT      DEFAULT 0 COMMENT '性别 (0:未设置, 1:男, 2:女)',
    `region`     VARCHAR(255) DEFAULT NULL COMMENT '地区',
    `birthday`   DATE         DEFAULT NULL COMMENT '生日',

    -- 冗余字段
    `followers_count` INT NOT NULL DEFAULT 0 COMMENT '粉丝数',
    `following_count` INT NOT NULL DEFAULT 0 COMMENT '关注数',
    PRIMARY KEY (`user_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户资料表';

INSERT INTO `user_detail` (`user_id`, `nickname`)
VALUES (10000, '管理员');
