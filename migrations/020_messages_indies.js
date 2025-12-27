db.messages.createIndex(
    { conversation_id: 1, created_at: -1 }, // 1: 升序, -1: 降序
    { name: "idx_conv_time", background: true }
);

db.messages.createIndex(
    { sender_id: 1 },
    { name: "idx_sender_id" }
);

db.runCommand({
    collMod: "messages",
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["conversation_id", "sender_id", "content", "media_type", "created_at"],
            properties: {
                conversation_id: { bsonType: "long" },
                sender_id: { bsonType: "long" },
                media_type: { enum: [0, 1, 2, 3] },
                created_at: { bsonType: "date" }
            }
        }
    },
    validationAction: "warn" // 发现校验不通过时，只发出警告，不阻止写入 (方便初期开发)
});