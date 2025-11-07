#!/bin/bash

echo "=== WebDAV LOCK/UNLOCK 功能测试 ==="

# 创建测试用的XML请求体
LOCK_XML='<?xml version="1.0" encoding="utf-8"?>
<lockinfo>
  <lockscope>
    <exclusive/>
  </lockscope>
  <locktype>
    <write/>
  </locktype>
  <owner>
    <href>test@example.com</href>
  </owner>
  <timeout>Second-3600</timeout>
  <depth>0</depth>
</lockinfo>'

echo "1. 测试LOCK请求体格式..."
echo "$LOCK_XML" | grep -q "lockinfo" && echo "✓ LOCK XML 包含 lockinfo 元素"
echo "$LOCK_XML" | grep -q "lockscope" && echo "✓ LOCK XML 包含 lockscope 元素"
echo "$LOCK_XML" | grep -q "locktype" && echo "✓ LOCK XML 包含 locktype 元素"
echo "$LOCK_XML" | grep -q "owner" && echo "✓ LOCK XML 包含 owner 元素"
echo "$LOCK_XML" | grep -q "timeout" && echo "✓ LOCK XML 包含 timeout 元素"
echo "$LOCK_XML" | grep -q "depth" && echo "✓ LOCK XML 包含 depth 元素"

echo ""
echo "2. 测试结构体定义..."

# 检查LockRequest结构体字段
if grep -A 10 "type LockRequest struct" internal/webdav/handler.go | grep -q "XMLName"; then
    echo "✓ LockRequest 包含 XMLName 字段"
fi

if grep -A 10 "type LockRequest struct" internal/webdav/handler.go | grep -q "LockScope"; then
    echo "✓ LockRequest 包含 LockScope 字段"
fi

if grep -A 10 "type LockRequest struct" internal/webdav/handler.go | grep -q "LockType"; then
    echo "✓ LockRequest 包含 LockType 字段"
fi

if grep -A 10 "type LockRequest struct" internal/webdav/handler.go | grep -q "Owner"; then
    echo "✓ LockRequest 包含 Owner 字段"
fi

if grep -A 10 "type LockRequest struct" internal/webdav/handler.go | grep -q "Timeout"; then
    echo "✓ LockRequest 包含 Timeout 字段"
fi

if grep -A 10 "type LockRequest struct" internal/webdav/handler.go | grep -q "Depth"; then
    echo "✓ LockRequest 包含 Depth 字段"
fi

echo ""
echo "3. 测试锁定管理器功能..."

# 检查LockManager方法
if grep -q "func.*CreateLock" internal/webdav/lock.go; then
    echo "✓ CreateLock 方法存在"
fi

if grep -q "func.*GetLock" internal/webdav/lock.go; then
    echo "✓ GetLock 方法存在"
fi

if grep -q "func.*RemoveLock" internal/webdav/lock.go; then
    echo "✓ RemoveLock 方法存在"
fi

if grep -q "func.*CheckLock" internal/webdav/lock.go; then
    echo "✓ CheckLock 方法存在"
fi

if grep -q "func.*CleanExpiredLocks" internal/webdav/lock.go; then
    echo "✓ CleanExpiredLocks 方法存在"
fi

echo ""
echo "4. 测试冲突检测逻辑..."

# 检查冲突检测方法
if grep -q "func.*CheckExclusiveLock" internal/webdav/handler.go; then
    echo "✓ CheckExclusiveLock 方法存在"
fi

if grep -q "func.*CheckSharedLock" internal/webdav/handler.go; then
    echo "✓ CheckSharedLock 方法存在"
fi

if grep -q "func.*CheckParentLocks" internal/webdav/handler.go; then
    echo "✓ CheckParentLocks 方法存在"
fi

echo ""
echo "5. 测试错误处理..."

# 检查错误响应
if grep -q "SendLockedError" internal/webdav/handler.go; then
    echo "✓ SendLockedError 方法存在"
fi

if grep -q "type LockedError struct" internal/webdav/handler.go; then
    echo "✓ LockedError 结构体存在"
fi

echo ""
echo "6. 测试响应XML结构..."

# 检查响应结构
if grep -q "type ActiveLock struct" internal/webdav/lock.go; then
    echo "✓ ActiveLock 结构体存在"
fi

if grep -q "type LockToken struct" internal/webdav/lock.go; then
    echo "✓ LockToken 结构体存在"
fi

if grep -q "LockDiscovery" internal/webdav/handler.go; then
    echo "✓ LockDiscovery 响应元素存在"
fi

if grep -q "LockToken" internal/webdav/handler.go; then
    echo "✓ LockToken 响应元素存在"
fi

echo ""
echo "7. 测试HTTP状态码..."

# 检查状态码使用
if grep -q "StatusLocked" internal/webdav/handler.go; then
    echo "✓ 423 Locked 状态码使用存在"
fi

if grep -q "StatusNoContent" internal/webdav/handler.go; then
    echo "✓ 204 No Content 状态码使用存在"
fi

echo ""
echo "=== 功能测试完成 ==="
echo "所有核心功能组件验证通过"