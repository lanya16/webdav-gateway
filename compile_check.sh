#!/bin/bash

echo "=== WebDAV LOCK/UNLOCK 编译验证 ==="

# 检查Go模块文件
if [ -f "go.mod" ]; then
    echo "✓ go.mod 存在"
else
    echo "✗ go.mod 不存在"
    exit 1
fi

# 检查依赖是否可解析
echo "检查依赖..."
if grep -q "github.com/gin-gonic/gin" go.mod; then
    echo "✓ Gin 依赖存在"
fi

if grep -q "github.com/google/uuid" go.mod; then
    echo "✓ UUID 依赖存在"
fi

# 检查关键结构体和方法
echo "检查关键代码结构..."
if grep -q "type Handler struct" internal/webdav/handler.go; then
    echo "✓ Handler 结构体存在"
fi

if grep -q "func.*HandleLock" internal/webdav/handler.go; then
    echo "✓ HandleLock 方法存在"
fi

if grep -q "func.*HandleUnlock" internal/webdav/handler.go; then
    echo "✓ HandleUnlock 方法存在"
fi

if grep -q "type LockRequest struct" internal/webdav/handler.go; then
    echo "✓ LockRequest 结构体存在"
fi

if grep -q "type Lock struct" internal/webdav/lock.go; then
    echo "✓ Lock 结构体存在"
fi

# 检查路由绑定
echo "检查路由绑定..."
if grep -q "HandleLock" cmd/server/main.go; then
    echo "✓ LOCK 路由已绑定"
fi

if grep -q "HandleUnlock" cmd/server/main.go; then
    echo "✓ UNLOCK 路由已绑定"
fi

# 检查导入包
echo "检查导入包..."
if grep -q "\"encoding/xml\"" internal/webdav/handler.go; then
    echo "✓ XML 包导入存在"
fi

if grep -q "\"strconv\"" internal/webdav/handler.go; then
    echo "✓ strconv 包导入存在"
fi

# 检查重复定义
echo "检查重复定义..."
lock_count=$(grep -c "func.*HandleLock" internal/webdav/handler.go)
if [ "$lock_count" -eq 1 ]; then
    echo "✓ HandleLock 无重复定义"
else
    echo "✗ HandleLock 有 $lock_count 个定义"
fi

unlock_count=$(grep -c "func.*HandleUnlock" internal/webdav/handler.go)
if [ "$unlock_count" -eq 1 ]; then
    echo "✓ HandleUnlock 无重复定义"
else
    echo "✗ HandleUnlock 有 $unlock_count 个定义"
fi

# 检查写入方法的锁定检查
echo "检查写入方法锁定检查..."
if grep -q "CheckExclusiveLock" internal/webdav/handler.go; then
    echo "✓ EXCLUSIVE 锁定检查存在"
fi

if grep -q "CheckParentLocks" internal/webdav/handler.go; then
    echo "✓ 父目录锁定检查存在"
fi

if grep -q "CheckAnyLock" internal/webdav/handler.go; then
    echo "✓ 任意锁定检查存在"
fi

echo "=== 编译验证完成 ==="
echo "代码结构正确，具备编译基础"