# Cornerstone 后端服务

## 项目简介

Cornerstone 是一个基于 Go 语言开发的现代化社交平台后端系统。

## 项目架构

本项目采用分层架构设计，主要包括以下组件：

- **API 层**: 使用 Gin 框架构建 RESTful API 接口
- **业务逻辑层**: 处理具体的业务逻辑
- **数据访问层**: 封装数据库操作
- **消息队列**: 基于 Kafka 的异步处理机制
- **搜索引擎**: 使用 Elasticsearch 提供全文检索能力
- **对象存储**: 使用 MinIO 存储媒体文件
- **缓存系统**: 使用 Redis 缓存热点数据
- **文档数据库**: 使用 MongoDB 存储非结构化数据

### 项目目录结构

```
.
├── cmd/
│   └── api/
│       └── main.go                 # 项目入口文件
├── configs/
│   └── config_template.yml         # 配置文件模板
├── internal/
│   ├── api/
│   │   ├── config/                # 配置管理相关
│   │   ├── dto/                   # 数据传输对象
│   │   ├── handler/               # HTTP请求处理器
│   │   ├── middleware/            # 中间件
│   │   ├── handlers_group.go      # Handler分组
│   │   └── route.go               # API路由定义
│   ├── job/                       # 定时任务
│   │   ├── media_clean_job.go     # 媒体清理任务
│   │   ├── post_comment_job.go    # 帖子评论任务
│   │   ├── post_metric_job.go     # 帖子指标任务
│   │   ├── user_interest_job.go   # 用户兴趣任务
│   │   └── user_metric_job.go     # 用户指标任务
│   ├── model/                     # 数据模型
│   │   ├── post.go                # 帖子模型
│   │   ├── user.go                # 用户模型
│   │   ├── comment.go             # 评论模型
│   │   └── ...                    # 其他模型文件
│   ├── pkg/                       # 通用工具包
│   │   ├── consts/                # 常量定义
│   │   ├── cron/                  # 定时任务管理
│   │   ├── database/              # 数据库连接管理
│   │   ├── es/                    # Elasticsearch相关
│   │   ├── kafka/                 # Kafka消息队列
│   │   ├── llm/                   # 大语言模型集成
│   │   ├── minio/                 # 对象存储
│   │   ├── mongo/                 # MongoDB操作
│   │   ├── redis/                 # Redis缓存
│   │   ├── response/              # 响应格式
│   │   ├── security/              # 安全相关
│   │   └── util/                  # 通用工具函数
│   ├── repository/                # 数据访问层
│   │   ├── post_repo.go           # 帖子数据访问
│   │   ├── user_repo.go           # 用户数据访问
│   │   └── ...                    # 其他数据访问文件
│   ├── service/                   # 业务逻辑层
│   │   ├── post_service.go        # 帖子业务逻辑
│   │   ├── user_service.go        # 用户业务逻辑
│   │   └── ...                    # 其他业务逻辑文件
│   └── wire/                      # 依赖注入
├── lib/                           # 第三方库
│   └── note.md                    # 库说明
├── migrations/                    # 数据库迁移脚本
│   ├── 000_pre.sql                # 预处理脚本
│   ├── 001_users.sql              # 用户表创建
│   ├── 002_user_detail.sql        # 用户详情表
│   └── ...                        # 其他迁移脚本
├── prompts/                       # LLM提示词
│   ├── chat.txt                   # 聊天提示词
│   ├── content-process.txt        # 内容处理提示词
│   └── ...                        # 其他提示词文件
├── .gitignore                     # Git忽略文件配置
├── README.md                      # 项目说明文档
├── go.mod                         # Go模块定义
└── go.sum                         # Go模块校验和
```

## 技术栈

### 主要技术

- **编程语言**: Go (Golang) 1.24+
- **Web 框架**: Gin
- **数据库**: MySQL (使用 GORM ORM)
- **缓存**: Redis
- **文档数据库**: MongoDB
- **对象存储**: MinIO
- **搜索引擎**: Elasticsearch
- **消息队列**: Kafka
- **配置管理**: Viper
- **日志系统**: slog, Logstash

### 核心依赖

- gin-gonic/gin: 轻量级 Web 框架
- gorm.io/gorm: ORM 库
- go-redis/redis: Redis 客户端
- elastic/go-elasticsearch: Elasticsearch 客户端
- IBM/sarama: Kafka 客户端
- minio/minio-go: MinIO 客户端
- spf13/viper: 配置管理
- golang-jwt/jwt: JWT 认证

### 外部库依赖

- FFmpeg: 媒体处理
- whisper: 音频处理

## 功能模块

### 1. 用户模块 (User Module)
- 用户注册/登录（支持手机号验证）
- 用户信息管理（昵称、头像、个人资料等）
- 密码找回/修改
- 用户角色权限管理
- 用户封禁/解封（管理员功能）
- 用户搜索功能

### 2. 内容管理模块 (Post Module)
- 发布/编辑/删除帖子
- 内容审核（自动+人工审核）
- 帖子推荐算法
- 内容搜索功能
- 帖子状态管理（审核中、已发布、已驳回等）

### 3. 社交互动模块 (Social Interaction)
- 点赞/收藏功能
- 评论/回复功能
- 关注/取消关注
- 举报功能
- 互动统计（点赞数、评论数、收藏数等）

### 4. 即时通讯模块 (IM Module)
- WebSocket 实时连接
- 私信聊天功能
- 聊天记录查询
- 会话列表管理
- 消息已读标记

### 5. 通知系统 (Notification System)
- 系统消息推送
- 未读消息计数
- 消息批量标记已读

### 6. 内容推荐模块 (Recommendation System)
- 基于用户兴趣的内容推荐
- 个性化推荐算法
- 内容标签系统

### 7. 统计分析模块 (Analytics Module)
- 用户行为分析（7天/30天数据）
- 内容表现分析
- 用户内容指标统计
- 帖子数据统计

### 8. AI 代理模块 (AI Agent)
- 智能对话功能
- 内容搜索功能
- LLM 集成（支持文本、视觉模型）

### 9. 媒体处理模块 (Media Processing)
- 图片/视频上传
- 文件格式验证
- 媒体文件存储到 MinIO
- 媒体内容审核

## 项目启动

### 环境准备

1. **安装依赖工具**
   - Go 1.24+
   - MySQL
   - Redis
   - MongoDB
   - Elasticsearch
   - Kafka
   - MinIO

2. **环境变量配置**

复制配置模板并填入实际配置：
```bash
cp configs/config_template.yml configs/config.yml
```

修改 `configs/config.yml` 中的各项配置：
- 数据库连接信息
- Redis 连接信息
- MongoDB 连接信息
- Elasticsearch 连接信息
- Kafka 连接信息
- MinIO 连接信息
- LLM 服务配置
- 短信服务配置

### 启动步骤

1. **下载依赖**
```bash
go mod tidy
```

2. **运行数据库迁移**
```bash
# 执行 SQL 和其他迁移脚本
```

3. **启动服务**
```bash
cd cmd/api
go run main.go
```

服务将启动在 `http://localhost:8080`

## 构建与部署脚本

项目提供了便捷的构建和部署脚本，位于 `script/` 目录下：

- `build.sh`: 用于构建不同平台的二进制文件
- `deploy.sh`: 用于本地环境检查和运行准备

### 构建脚本

```bash
# 执行构建脚本，生成多平台二进制文件
chmod +x script/build.sh
./script/build.sh
```

### 本地运行脚本

```bash
# 检查运行环境并准备配置
chmod +x script/deploy.sh
./script/deploy.sh
```

## 项目特点

- **高可用性**: 使用多种数据库和缓存系统保证数据可靠性
- **高性能**: 异步处理机制和缓存策略提升系统性能
- **可扩展性**: 微服务架构便于功能扩展
- **安全性**: JWT 认证、内容审核、防刷机制
- **智能化**: 集成 LLM 提供智能服务
- **实时性**: WebSocket 支持即时通讯

## 开发规范

- 代码风格遵循 Go 官方规范
- 使用统一的错误处理机制
- 采用 DTO 模式进行数据传输
- 使用中间件处理认证和授权
- 日志记录遵循结构化日志规范

## 许可证

本项目采用 MIT 许可证。