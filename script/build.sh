#!/bin/bash

# Cornerstone 项目构建脚本

set -e

# 设置项目名称和输出目录
PROJECT_NAME="cornerstone"
OUTPUT_DIR="./bin"

# 创建输出目录
mkdir -p ${OUTPUT_DIR}

# 获取当前 Git 提交的 SHA
GIT_COMMIT=$(git rev-parse HEAD)
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "开始构建 ${PROJECT_NAME}..."
echo "Git Commit: ${GIT_COMMIT}"
echo "Build Time: ${BUILD_TIME}"

# 设置编译参数
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build \
    -ldflags "-X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o ${OUTPUT_DIR}/${PROJECT_NAME}-linux-amd64 \
    ./cmd/api/main.go

echo "Linux amd64 版本构建完成"

# 构建 Windows 版本
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
    go build \
    -ldflags "-X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o ${OUTPUT_DIR}/${PROJECT_NAME}-windows-amd64.exe \
    ./cmd/api/main.go

echo "Windows amd64 版本构建完成"

# 构建 macOS 版本
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 \
    go build \
    -ldflags "-X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o ${OUTPUT_DIR}/${PROJECT_NAME}-darwin-amd64 \
    ./cmd/api/main.go

echo "macOS amd64 版本构建完成"

# 构建 macOS ARM64 版本
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
    go build \
    -ldflags "-X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o ${OUTPUT_DIR}/${PROJECT_NAME}-darwin-arm64 \
    ./cmd/api/main.go

echo "macOS arm64 版本构建完成"

echo "所有平台构建完成，输出文件位于 ${OUTPUT_DIR}/ 目录"