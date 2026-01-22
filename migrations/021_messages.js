/**
 * MongoDB 消息集合初始化脚本
 */

// 定义数据库名变量
const dbName = 'Cornerstone';
const collName = 'messages';

// 获取数据库引用
const currentDb = db.getSiblingDB(dbName);

// 如果集合已存在，则删除
currentDb.getCollection(collName).drop();

// 创建集合并配置严格的 JSON Schema 校验
currentDb.createCollection(collName, {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["conversation_id", "sender_id", "msg_type", "content", "seq", "created_at"],
            properties: {
                conversation_id: {
                    bsonType: "long",
                    description: "关联MySQL的会话ID，必须为Long类型"
                },
                sender_id: {
                    bsonType: "long",
                    description: "发送者用户ID"
                },
                msg_type: {
                    bsonType: "int",
                    enum: [1, 2, 3],
                    description: "消息类型: 1-正常消息, 2-音频消息, 3-撤回提醒"
                },
                content: {
                    bsonType: "string",
                    description: "文本内容或消息预览"
                },
                payload: {
                    bsonType: "array",
                    description: "结构化媒体信息 (可选)",
                    properties: {
                        url: { bsonType: "string" },
                        width: { bsonType: "int" },
                        height: { bsonType: "int" },
                        duration: { bsonType: "double" },
                        mime_type: { bsonType: "string" },
                        cover_url: { bsonType: "string" }
                    }
                },
                seq: {
                    bsonType: "long",
                    description: "会话内唯一有序序号"
                },
                reply_to: {
                    bsonType: "long"
                },
                created_at: {
                    bsonType: "date"
                }
            }
        }
    },
    validationAction: "error", // 严格拦截
    validationLevel: "strict"
});

// 创建索引
const collection = currentDb.getCollection(collName);

// 索引一：支撑聊天记录分页加载
collection.createIndex(
    { conversation_id: 1, seq: -1 },
    { name: "idx_conv_seq", background: true }
);

// 索引二：支撑用户发送记录检索
collection.createIndex(
    { sender_id: 1 },
    { name: "idx_sender", background: true }
);

print(">>> " + dbName + "." + collName + " 数据库初始化完成！");