CREATE TABLE `post_media`
(
    `id`           BIGINT       NOT NULL AUTO_INCREMENT COMMENT '媒体记录ID',
    `post_id`      BIGINT       NOT NULL COMMENT '关联的笔记ID (逻辑关联 posts.id)',
    -- 常见值: image/jpeg, image/png, image/webp, video/mp4, video/quicktime
    `file_type`    VARCHAR(64)  NOT NULL COMMENT '文件MIME类型 (e.g., image/jpeg, video/mp4)',
    `media_url`    VARCHAR(512) NOT NULL COMMENT '媒体的OSS存储URL',
    `sort_order`   TINYINT      NOT NULL DEFAULT 0 COMMENT '媒体在笔记中的显示顺序 (0, 1, 2...)',
    `width`        INT          NOT NULL DEFAULT 0 COMMENT '原始素材宽度 (px)',
    `height`       INT          NOT NULL DEFAULT 0 COMMENT '原始素材高度 (px)',
    `cover_url`    VARCHAR(512) DEFAULT NULL COMMENT '视频封面或图片缩略图URL',
    `created_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_post_id_sort` (`post_id`, `sort_order`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='笔记媒体表 (图文视频混合)';