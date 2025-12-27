CREATE TABLE `tags`
(
    `id`          BIGINT       NOT NULL AUTO_INCREMENT COMMENT '标签ID',
    `name`        VARCHAR(50)  NOT NULL COMMENT '标签名称',
    `description` VARCHAR(255) DEFAULT NULL COMMENT '标签描述',
    `is_main`     TINYINT(1)   DEFAULT 0 COMMENT '是否为主要标签',
    `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tag_name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='标签定义表 (中央词典)';

INSERT INTO `tags` (name, description, is_main, created_at)
VALUES
('编程开发', '代码改变世界，技术交流与分享', 1, NOW()),
('科技数码', '探索前沿黑科技，评测数码新品', 1, NOW()),
('互联网', '关注互联网趋势，产品与运营方法论', 1, NOW()),
('美食探店', '唯有爱与美食不可辜负', 1, NOW()),
('旅行摄影', '记录美好瞬间，探索诗与远方', 1, NOW()),
('时尚穿搭', '潮流风向标，做最靓的自己', 1, NOW()),
('萌宠生活', '治愈系猫狗日常，云吸宠基地', 1, NOW()),
('游戏电竞', '硬核玩家集合，攻略与赛事', 1, NOW()),
('影视综艺', '热门剧集讨论，经典电影推荐', 1, NOW()),
('二次元', '动漫番剧，Cosplay与手办模型', 1, NOW()),
('运动健身', '自律给我自由，燃烧你的卡路里', 1, NOW()),
('职场成长', '升职加薪攻略，职场生存指南', 1, NOW());