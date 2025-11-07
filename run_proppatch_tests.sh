#!/bin/bash

echo "==========================================="
echo "PROPPATCH 重构后单元测试运行脚本"
echo "==========================================="

# 设置环境变量
export GO111MODULE=on

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "错误：Go编译器未安装"
    echo "请先安装Go编译器："
    echo "  curl -L https://go.dev/dl/go1.21.5.linux-amd64.tar.gz -o /tmp/go.tar.gz"
    echo "  cd /tmp && tar -xzf go.tar.gz"
    echo "  export PATH=\$PATH:/tmp/go/bin"
    exit 1
fi

echo "Go版本信息："
go version

echo ""
echo "开始运行测试..."

# 进入项目目录
cd /workspace/webdav-gateway

# 运行测试
echo ""
echo "1. 运行验证器测试..."
go test -v ./internal/webdav/validators/

echo ""
echo "2. 运行字符串工具测试..."
go test -v ./internal/webdav/utils/

echo ""
echo "3. 运行XML序列化测试..."
go test -v ./internal/webdav/xml/

echo ""
echo "4. 运行SQL构建器测试..."
go test -v ./internal/webdav/sql_builder_test.go

echo ""
echo "5. 运行XML处理器测试..."
go test -v ./internal/webdav/xml_proppatch_test.go

echo ""
echo "6. 运行属性服务测试..."
go test -v ./internal/webdav/property_service_test.go

echo ""
echo "7. 运行处理器集成测试..."
go test -v ./internal/webdav/handler_test.go

echo ""
echo "8. 运行整个webdav包的测试..."
go test -v ./internal/webdav/...

echo ""
echo "9. 生成测试覆盖率报告..."
go test -coverprofile=coverage.out ./internal/webdav/...
go tool cover -html=coverage.out -o coverage.html

echo ""
echo "10. 查看覆盖率摘要..."
go tool cover -func=coverage.out

echo ""
echo "==========================================="
echo "测试完成！覆盖率报告生成在："
echo "  - coverage.out (原始数据)"
echo "  - coverage.html (HTML报告)"
echo "==========================================="