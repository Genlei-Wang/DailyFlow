#!/bin/bash

# DailyFlow 构建脚本
# 用于在 Linux/macOS 上交叉编译 Windows 可执行文件

set -e

echo "========================================"
echo "  DailyFlow Build Script"
echo "========================================"

# 设置变量
APP_NAME="DailyFlow"
VERSION="1.0.0"
OUTPUT_DIR="dist"
OUTPUT_FILE="${OUTPUT_DIR}/${APP_NAME}.exe"

# 清理旧的构建
echo "清理旧的构建文件..."
rm -rf ${OUTPUT_DIR}
mkdir -p ${OUTPUT_DIR}

# 设置环境变量
export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=1

# 检查是否安装了 MinGW-w64
if ! command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    echo "错误: 未找到 x86_64-w64-mingw32-gcc"
    echo "请安装 MinGW-w64:"
    echo "  Ubuntu/Debian: sudo apt-get install gcc-mingw-w64"
    echo "  macOS: brew install mingw-w64"
    exit 1
fi

export CC=x86_64-w64-mingw32-gcc
export CXX=x86_64-w64-mingw32-g++

echo "构建目标: ${GOOS}/${GOARCH}"
echo "CGO 编译器: ${CC}"

# 构建
echo "开始构建..."
go build \
    -ldflags="-s -w -H windowsgui" \
    -o ${OUTPUT_FILE} \
    ./cmd/dailyflow

# 检查构建结果
if [ -f ${OUTPUT_FILE} ]; then
    FILE_SIZE=$(du -h ${OUTPUT_FILE} | cut -f1)
    echo "========================================"
    echo "  构建成功！"
    echo "========================================"
    echo "输出文件: ${OUTPUT_FILE}"
    echo "文件大小: ${FILE_SIZE}"
    
    # 检查文件大小限制（应小于 5MB）
    FILE_SIZE_BYTES=$(stat -f%z ${OUTPUT_FILE} 2>/dev/null || stat -c%s ${OUTPUT_FILE} 2>/dev/null)
    MAX_SIZE=$((5 * 1024 * 1024))  # 5MB
    
    if [ ${FILE_SIZE_BYTES} -gt ${MAX_SIZE} ]; then
        echo "警告: 文件大小超过 5MB 限制！"
    fi
else
    echo "========================================"
    echo "  构建失败！"
    echo "========================================"
    exit 1
fi

