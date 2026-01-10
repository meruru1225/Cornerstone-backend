CREATE TABLE `posts`
(
    `id`              BIGINT     NOT NULL AUTO_INCREMENT COMMENT '笔记ID',
    `user_id`         BIGINT     NOT NULL COMMENT '作者ID',
    `title`           VARCHAR(255)        DEFAULT NULL COMMENT '笔记标题',
    `content`         TEXT       NOT NULL COMMENT '笔记正文',
    `media_list`      JSON                DEFAULT NULL COMMENT '笔记附带内容',
    `likes_count`     INT        NOT NULL DEFAULT 0 COMMENT '点赞数',
    `comments_count`  INT        NOT NULL DEFAULT 0 COMMENT '评论数',
    `collects_count`  INT        NOT NULL DEFAULT 0 COMMENT '收藏数',
    `views_count`      INT        NOT NULL DEFAULT 0 COMMENT '浏览数',
    `status`          TINYINT    NOT NULL DEFAULT 0 COMMENT '状态(0:审核中, 1:已发布, 2:拒绝, 3:待人工)',
    `is_deleted`      TINYINT(1) NOT NULL DEFAULT 0 COMMENT '逻辑删除标志 (0:否, 1:是)',
    `created_at`      DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='笔记主表';

/*
  =============================================================================
  ARCHIVED SCHEMA: post_media (DEPRECATED)
  =============================================================================

  【重构说明】：
  该表被废弃，其功能已由 `posts.media_list` (JSON) 字段替代。

  【原始结构存档】：
  CREATE TABLE `post_media` (
      `id`           BIGINT       NOT NULL AUTO_INCREMENT COMMENT '媒体记录ID',
      `post_id`      BIGINT       NOT NULL COMMENT '关联的笔记ID (逻辑关联 posts.id)',
      `file_type`    VARCHAR(64)  NOT NULL COMMENT '文件MIME类型',
      `media_url`    VARCHAR(512) NOT NULL COMMENT '媒体存储URL',
      `sort_order`   TINYINT      NOT NULL DEFAULT 0 COMMENT '显示顺序',
      `width`        INT          NOT NULL DEFAULT 0 COMMENT '宽度',
      `height`       INT          NOT NULL DEFAULT 0 COMMENT '高度',
      `duration`     INT          NOT NULL DEFAULT 0 COMMENT '视频时长',
      `cover_url`    VARCHAR(512) DEFAULT NULL COMMENT '封面图',
      `created_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
      `updated_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (`id`),
      KEY `idx_post_id_sort` (`post_id`, `sort_order`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;
  =============================================================================
*/