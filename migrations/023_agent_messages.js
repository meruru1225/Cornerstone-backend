/**
 * MongoDB Agent 消息集合初始化脚本
 */

const dbName = 'Cornerstone';
const collName = 'agent_messages';

const currentDb = db.getSiblingDB(dbName);

// 如果集合已存在，则删除
currentDb.getCollection(collName).drop();

// 创建集合：只保留核心业务字段
currentDb.createCollection(collName, {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["conversation_id", "sender_id", "content", "created_at"],
            properties: {
                conversation_id: {
                    bsonType: "string", // UUID 字符串
                    description: "会话唯一标识(UUID)"
                },
                sender_id: {
                    bsonType: "long", // 0:Agent, 1:Guest, 1001+:User
                    description: "发送者ID"
                },
                content: {
                    bsonType: "string",
                    description: "对话内容"
                },
                created_at: {
                    bsonType: "date",
                    description: "消息创建时间"
                }
            }
        }
    },
    validationAction: "error",
    validationLevel: "strict"
});

const collection = currentDb.getCollection(collName);

/**
 * 核心索引：支撑按会话拉取历史记录
 * 使用 conversation_id 分组，并按 created_at 倒序排列获取最新消息
 */
collection.createIndex(
    { conversation_id: 1, created_at: -1 },
    { name: "idx_agent_conv_time", background: true }
);

/**
 * 过期索引：支撑消息过期删除
 * 使用 created_at 字段，设置过期时间为30天
 */
collection.createIndex(
    { "created_at": 1 },
    { expireAfterSeconds: 2592000 } // 30天 = 30 * 24 * 3600 秒
);

print(">>> " + dbName + "." + collName + " Agent Message初始化完成！");