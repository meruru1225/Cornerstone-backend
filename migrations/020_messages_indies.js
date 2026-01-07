db.messages.createIndex(
    { conversation_id: 1, seq: -1 },
    { name: "idx_conv_seq", background: true }
);

db.messages.createIndex(
    { sender_id: 1 },
    { name: "idx_sender" }
);

db.runCommand({
    collMod: "messages",
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["conversation_id", "sender_id", "content", "media_type", "seq", "created_at"],
            properties: {
                conversation_id: { bsonType: "long", description: "关联MySQL的会话ID" },
                sender_id: { bsonType: "long", description: "发送者用户ID" },
                content: { bsonType: "string", description: "消息内容" },
                media_type: {
                    bsonType: "string",
                    description: "消息媒体类型: text, image, video, audio, file"
                },
                seq: { bsonType: "long", description: "该消息在会话中的唯一有序序号" },
                created_at: { bsonType: "date" }
            }
        }
    },
    validationAction: "warn"
});