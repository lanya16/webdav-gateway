#!/bin/bash

# API测试脚本

BASE_URL="http://localhost:8080"
TOKEN=""

echo "======================================"
echo "WebDAV Gateway API 测试"
echo "======================================"
echo ""

# 1. 健康检查
echo "1. 健康检查测试..."
curl -s "${BASE_URL}/health" | jq '.'
echo ""

# 2. 用户注册
echo "2. 用户注册测试..."
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123",
    "display_name": "Test User"
  }')
echo $REGISTER_RESPONSE | jq '.'
echo ""

# 3. 用户登录
echo "3. 用户登录测试..."
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }')
echo $LOGIN_RESPONSE | jq '.'

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.token')
echo "获取到Token: ${TOKEN:0:20}..."
echo ""

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "登录失败，无法继续测试"
  exit 1
fi

# 4. 获取用户信息
echo "4. 获取用户信息测试..."
curl -s -X GET "${BASE_URL}/api/auth/me" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# 5. 上传文件
echo "5. 上传文件测试..."
echo "Hello WebDAV!" > /tmp/test.txt
curl -s -X PUT "${BASE_URL}/webdav/test.txt" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: text/plain" \
  --data-binary "@/tmp/test.txt"
echo "上传完成"
echo ""

# 6. PROPFIND - 列出文件
echo "6. PROPFIND测试..."
curl -s -X PROPFIND "${BASE_URL}/webdav/" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?>
<propfind xmlns="DAV:">
  <prop>
    <displayname/>
    <getcontentlength/>
    <getcontenttype/>
  </prop>
</propfind>' | xmllint --format -
echo ""

# 7. 下载文件
echo "7. 下载文件测试..."
curl -s -X GET "${BASE_URL}/webdav/test.txt" \
  -H "Authorization: Bearer $TOKEN"
echo ""
echo ""

# 8. 创建目录
echo "8. 创建目录测试..."
curl -s -X MKCOL "${BASE_URL}/webdav/testfolder" \
  -H "Authorization: Bearer $TOKEN"
echo "创建目录完成"
echo ""

# 9. 创建分享
echo "9. 创建分享测试..."
SHARE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/shares" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "file_path": "/test.txt",
    "share_name": "测试文件分享",
    "password": "share123",
    "expires_in": 168,
    "max_downloads": 10
  }')
echo $SHARE_RESPONSE | jq '.'

SHARE_TOKEN=$(echo $SHARE_RESPONSE | jq -r '.share_token')
echo "分享Token: $SHARE_TOKEN"
echo ""

# 10. 获取分享信息
echo "10. 获取分享信息测试..."
curl -s -X GET "${BASE_URL}/share/${SHARE_TOKEN}" | jq '.'
echo ""

# 11. 访问分享
echo "11. 访问分享测试..."
curl -s -X POST "${BASE_URL}/share/${SHARE_TOKEN}/access" \
  -H "Content-Type: application/json" \
  -d '{
    "password": "share123"
  }' | jq '.'
echo ""

# 12. 列出分享
echo "12. 列出我的分享测试..."
curl -s -X GET "${BASE_URL}/api/shares" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# 13. 移动文件
echo "13. 移动文件测试..."
curl -s -X MOVE "${BASE_URL}/webdav/test.txt" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Destination: ${BASE_URL}/webdav/testfolder/test.txt" \
  -H "Overwrite: T"
echo "移动文件完成"
echo ""

# 14. 复制文件
echo "14. 复制文件测试..."
curl -s -X COPY "${BASE_URL}/webdav/testfolder/test.txt" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Destination: ${BASE_URL}/webdav/test_copy.txt" \
  -H "Overwrite: T"
echo "复制文件完成"
echo ""

# 15. 删除文件
echo "15. 删除文件测试..."
curl -s -X DELETE "${BASE_URL}/webdav/test_copy.txt" \
  -H "Authorization: Bearer $TOKEN"
echo "删除文件完成"
echo ""

echo "======================================"
echo "所有测试完成！"
echo "======================================"

# 清理
rm -f /tmp/test.txt