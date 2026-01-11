/**
 * MongoDB 系统通知集合 (sys_box) 初始化脚本
 */

// 定义数据库名变量
const dbName = 'Cornerstone';
const collName = 'sys_box';

// 获取数据库引用
const currentDb = db.getSiblingDB(dbName);

// 如果集合已存在，则删除 (生产环境慎用，初始化环境可用)
currentDb.getCollection(collName).drop();

// 创建集合并配置严格的 JSON Schema 校验
currentDb.createCollection(collName, {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["receiver_id", "sender_id", "type", "is_read", "created_at"],
            properties: {
                receiver_id: {
                    bsonType: "long",
                    description: "消息接收者ID"
                },
                sender_id: {
                    bsonType: "long",
                    description: "动作发起者ID (系统通知可为0)"
                },
                type: {
                    bsonType: "int",
                    enum: [1, 2, 3, 4, 5],
                    description: "通知类型: 1-帖子点赞, 2-帖子收藏, 3-帖子评论, 4-评论点赞, 5-被关注"
                },
                target_id: {
                    bsonType: "long",
                    description: "关联的目标ID (如帖子ID、评论ID)"
                },
                content: {
                    bsonType: "string",
                    description: "通知文案预览或评论片段"
                },
                payload: {
                    bsonType: "object",
                    description: "额外元数据 (可选)",
                    properties: {
                        post_title: { bsonType: "string" }, // 帖子标题预览
                        comment_id: { bsonType: "long" },   // 关联的评论ID
                        avatar: { bsonType: "string" }      // 发起人头像快照
                    }
                },
                is_read: {
                    bsonType: "bool",
                    description: "是否已读: false-未读, true-已读"
                },
                created_at: {
                    bsonType: "date",
                    description: "创建时间"
                }
            }
        }
    },
    validationAction: "error",
    validationLevel: "strict"
});

// 创建索引
const collection = currentDb.getCollection(collName);

// 索引一：支撑用户收信箱列表查询 (按时间倒序)
collection.createIndex(
    { receiver_id: 1, created_at: -1 },
    { name: "idx_receiver_time", background: true }
);

// 索引二：支撑未读数红点查询
collection.createIndex(
    { receiver_id: 1, is_read: 1 },
    { name: "idx_unread_count", background: true }
);

// 索引三：防止重复通知 (可选)
// 例如：同一个用户对同一个帖子短时间内重复点赞不产生多条通知
collection.createIndex(
    { receiver_id: 1, sender_id: 1, type: 1, target_id: 1 },
    { name: "idx_unique_notify", background: true }
);

print(">>> " + dbName + "." + collName + " 数据库初始化完成！");