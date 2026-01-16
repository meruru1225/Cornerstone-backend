#!/bin/bash

# Cornerstone 项目本地运行脚本

set -e

echo "==================================="
echo "Cornerstone 项目本地运行脚本"
echo "==================================="

# 检查必要工具
MISSING_TOOLS=()

if ! command -v go &> /dev/null; then
    MISSING_TOOLS+=("go")
fi

if [ ${#MISSING_TOOLS[@]} -gt 0 ]; then
    echo "错误: 以下必要工具未找到: ${MISSING_TOOLS[*]}"
    echo "请先安装这些工具后再运行此脚本"
    exit 1
fi

echo "正在准备运行环境..."

# 确保依赖是最新的
echo "同步依赖..."
go mod tidy

# 检查配置文件是否存在
if [ ! -f "configs/config.yml" ]; then
    if [ -f "configs/config_template.yml" ]; then
        echo "创建配置文件..."
        cp configs/config_template.yml configs/config.yml
        echo "请编辑 configs/config.yml 文件以配置您的环境变量"
    else
        echo "错误: 配置文件不存在"
        exit 1
    fi
fi

# 运行数据库迁移
if [ -d "migrations" ]; then
    echo "请确保已执行数据库迁移脚本"
    echo "例如: mysql -u root -p < migrations/001_users.sql"
fi

echo "==================================="
echo "环境检查完成"
echo "请按以下步骤启动服务:"
echo "1. 确保 MySQL, Redis, MongoDB, Elasticsearch, Kafka, MinIO 服务正在运行"
echo "2. 编辑 configs/config.yml 文件以配置连接信息"
echo "3. 在项目根目录执行: go run cmd/api/main.go"
echo "==================================="

go run ../cmd/api/main.go