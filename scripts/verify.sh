#!/bin/bash

# 自动化验证脚本
# 在有Go和Docker的环境中运行此脚本进行完整验证

set -e  # 遇到错误立即退出

echo "======================================"
echo "WebDAV Gateway 自动化验证脚本"
echo "======================================"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查函数
check_command() {
    if command -v $1 &> /dev/null; then
        echo -e "${GREEN}✓${NC} $1 已安装"
        return 0
    else
        echo -e "${RED}✗${NC} $1 未安装"
        return 1
    fi
}

# 测试函数
run_test() {
    local test_name=$1
    local test_command=$2
    
    echo ""
    echo -e "${YELLOW}→${NC} 测试: $test_name"
    
    if eval $test_command; then
        echo -e "${GREEN}✓${NC} 通过: $test_name"
        return 0
    else
        echo -e "${RED}✗${NC} 失败: $test_name"
        return 1
    fi
}

FAILED_TESTS=0

echo "步骤 1: 检查环境"
echo "-----------------------------------"

check_command "go" || ((FAILED_TESTS++))
check_command "docker" || ((FAILED_TESTS++))
check_command "docker-compose" || ((FAILED_TESTS++))
check_command "curl" || ((FAILED_TESTS++))

if [ $FAILED_TESTS -gt 0 ]; then
    echo ""
    echo -e "${RED}环境检查失败，请安装缺失的工具${NC}"
    exit 1
fi

echo ""
echo "步骤 2: Go模块验证"
echo "-----------------------------------"

run_test "下载依赖" "go mod download" || ((FAILED_TESTS++))
run_test "整理依赖" "go mod tidy" || ((FAILED_TESTS++))
run_test "验证依赖" "go mod verify" || ((FAILED_TESTS++))

echo ""
echo "步骤 3: 代码编译"
echo "-----------------------------------"

run_test "编译代码" "go build -o bin/webdav-gateway ./cmd/server" || ((FAILED_TESTS++))
run_test "检查可执行文件" "test -f bin/webdav-gateway" || ((FAILED_TESTS++))

echo ""
echo "步骤 4: Docker构建"
echo "-----------------------------------"

run_test "构建Docker镜像" "docker build -t webdav-gateway:test -f deployments/docker/Dockerfile ." || ((FAILED_TESTS++))

echo ""
echo "步骤 5: 启动服务"
echo "-----------------------------------"

cd deployments/docker

run_test "停止旧服务" "docker-compose down -v" || true
run_test "启动服务" "docker-compose up -d" || ((FAILED_TESTS++))

echo "等待服务启动..."
sleep 30

run_test "检查容器状态" "docker-compose ps | grep -q 'Up'" || ((FAILED_TESTS++))

echo ""
echo "步骤 6: 健康检查"
echo "-----------------------------------"

run_test "应用健康检查" "curl -f http://localhost:8080/health" || ((FAILED_TESTS++))
run_test "MinIO健康检查" "curl -f http://localhost:9000/minio/health/live" || ((FAILED_TESTS++))

echo ""
echo "步骤 7: API功能测试"
echo "-----------------------------------"

cd ../..

# 用户注册
run_test "用户注册" "curl -f -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{\"username\":\"testuser\",\"email\":\"test@example.com\",\"password\":\"password123\"}'" || ((FAILED_TESTS++))

# 用户登录并获取Token
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"testuser","password":"password123"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗${NC} 失败: 获取Token"
    ((FAILED_TESTS++))
else
    echo -e "${GREEN}✓${NC} 通过: 获取Token"
    
    # 获取用户信息
    run_test "获取用户信息" "curl -f -X GET http://localhost:8080/api/auth/me \
      -H 'Authorization: Bearer $TOKEN'" || ((FAILED_TESTS++))
    
    # 上传文件
    echo "Hello WebDAV!" > /tmp/test_verification.txt
    run_test "上传文件" "curl -f -X PUT http://localhost:8080/webdav/test_verification.txt \
      -H 'Authorization: Bearer $TOKEN' \
      -H 'Content-Type: text/plain' \
      --data-binary '@/tmp/test_verification.txt'" || ((FAILED_TESTS++))
    
    # 下载文件
    run_test "下载文件" "curl -f -X GET http://localhost:8080/webdav/test_verification.txt \
      -H 'Authorization: Bearer $TOKEN'" || ((FAILED_TESTS++))
    
    # 创建目录
    run_test "创建目录" "curl -f -X MKCOL http://localhost:8080/webdav/test_folder \
      -H 'Authorization: Bearer $TOKEN'" || ((FAILED_TESTS++))
    
    # 创建分享
    run_test "创建分享" "curl -f -X POST http://localhost:8080/api/shares \
      -H 'Authorization: Bearer $TOKEN' \
      -H 'Content-Type: application/json' \
      -d '{\"file_path\":\"/test_verification.txt\",\"expires_in\":168}'" || ((FAILED_TESTS++))
    
    # 列出分享
    run_test "列出分享" "curl -f -X GET http://localhost:8080/api/shares \
      -H 'Authorization: Bearer $TOKEN'" || ((FAILED_TESTS++))
    
    # 清理
    rm -f /tmp/test_verification.txt
fi

echo ""
echo "步骤 8: 日志检查"
echo "-----------------------------------"

cd deployments/docker

echo "最近的应用日志:"
docker-compose logs --tail=20 webdav-gateway

echo ""
echo "======================================"
echo "验证完成"
echo "======================================"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}所有测试通过! ✓${NC}"
    echo ""
    echo "服务运行中:"
    echo "  WebDAV Gateway: http://localhost:8080"
    echo "  MinIO Console:  http://localhost:9001"
    echo ""
    echo "停止服务: cd deployments/docker && docker-compose down"
    exit 0
else
    echo -e "${RED}有 $FAILED_TESTS 个测试失败 ✗${NC}"
    echo ""
    echo "查看日志: cd deployments/docker && docker-compose logs"
    echo "停止服务: cd deployments/docker && docker-compose down"
    exit 1
fi