CREATE TABLE `tags`
(
    `id`          BIGINT       NOT NULL AUTO_INCREMENT COMMENT '标签ID',
    `name`        VARCHAR(50)  NOT NULL COMMENT '标签名称',
    `description` VARCHAR(255) DEFAULT NULL COMMENT '标签描述',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tag_name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='标签定义表 (中央词典)';

INSERT INTO `tags` (name, description)
VALUES
('编程开发', '代码改变世界，技术交流与分享'),
('科技数码', '探索前沿黑科技，评测数码新品'),
('互联网', '关注互联网趋势，产品与运营方法论'),
('美食探店', '唯有爱与美食不可辜负'),
('旅行摄影', '记录美好瞬间，探索诗与远方'),
('时尚穿搭', '潮流风向标，做最靓的自己'),
('萌宠生活', '治愈系猫狗日常，云吸宠基地'),
('游戏电竞', '硬核玩家集合，攻略与赛事'),
('影视综艺', '热门剧集讨论，经典电影推荐'),
('二次元', '动漫番剧，Cosplay与手办模型'),
('运动健身', '自律给我自由，燃烧你的卡路里'),
('职场成长', '升职加薪攻略，职场生存指南'),
('其他内容', '其他内容');