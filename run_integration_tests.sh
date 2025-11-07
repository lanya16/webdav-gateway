#!/bin/bash

echo "============================================="
echo "PROPPATCH 集成测试运行脚本"
echo "============================================="

# 设置环境变量
export GO111MODULE=on

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "❌ 错误：Go编译器未安装"
    echo "请先安装Go编译器："
    echo "  curl -L https://go.dev/dl/go1.21.5.linux-amd64.tar.gz -o /tmp/go.tar.gz"
    echo "  cd /tmp && tar -xzf go.tar.gz"
    echo "  export PATH=\$PATH:/tmp/go/bin"
    exit 1
fi

echo "✅ Go版本信息："
go version

echo ""
echo "开始运行PROPPATCH集成测试..."

# 进入项目目录
cd /workspace/webdav-gateway

# 创建测试报告目录
mkdir -p test_results/integration

echo ""
echo "============================================="
echo "1. 运行完整的单元测试套件"
echo "============================================="
echo "运行所有单元测试..."
go test -v ./internal/webdav/... -coverprofile=test_results/unit_coverage.out -covermode=atomic

echo ""
echo "生成单元测试覆盖率报告..."
go tool cover -func=test_results/unit_coverage.out > test_results/unit_coverage_summary.txt

echo ""
echo "============================================="
echo "2. 运行专门的集成测试"
echo "============================================="
echo "运行端到端集成测试..."
go test -v -run TestCompleteProppatchWorkflow ./internal/webdav/
go test -v -run TestConcurrentProppatchRequests ./internal/webdav/
go test -v -run TestProppatchTransactionRollback ./internal/webdav/
go test -v -run TestProppatchWithComplexXML ./internal/webdav/
go test -v -run TestProppatchErrorScenarios ./internal/webdav/

echo ""
echo "============================================="
echo "3. 运行压力和性能测试"
echo "============================================="
echo "运行压力测试（包含在完整测试中）..."
go test -v -run TestProppatchStressTest ./internal/webdav/
go test -v -run TestProppatchLargeScaleUpdate ./internal/webdav/
go test -v -run TestProppatchMemoryUsage ./internal/webdav/
go test -v -run TestProppatchPerformanceRegression ./internal/webdav/

echo ""
echo "============================================="
echo "4. 运行基准性能测试"
echo "============================================="
echo "运行基准性能测试..."
go test -bench=BenchmarkProppatch -benchmem -benchtime=5s ./internal/webdav/ > test_results/benchmark_results.txt

echo ""
echo "============================================="
echo "5. 运行短测试模式（快速验证）"
echo "============================================="
echo "运行快速集成测试..."
go test -v -run TestCompleteProppatchWorkflow ./internal/webdav/ -timeout=30s
go test -v -run TestConcurrentProppatchRequests ./internal/webdav/ -timeout=30s
go test -v -run TestProppatchWithComplexXML ./internal/webdav/ -timeout=30s

echo ""
echo "============================================="
echo "6. 生成综合测试报告"
echo "============================================="

# 生成测试报告
echo "生成HTML覆盖率报告..."
go tool cover -html=test_results/unit_coverage.out -o test_results/integration/coverage.html

# 运行所有测试并生成详细报告
echo "运行完整测试套件并生成报告..."
go test -v -race -coverprofile=test_results/race_coverage.out ./internal/webdav/... 2>&1 | tee test_results/integration/test_output.log

echo ""
echo "============================================="
echo "7. 测试结果摘要"
echo "============================================="

# 显示覆盖率摘要
echo "单元测试覆盖率摘要："
cat test_results/unit_coverage_summary.txt

echo ""
echo "基准测试结果摘要："
if [ -f test_results/benchmark_results.txt ]; then
    head -20 test_results/benchmark_results.txt
fi

echo ""
echo "集成测试日志摘要："
if [ -f test_results/integration/test_output.log ]; then
    echo "测试总执行时间："
    tail -10 test_results/integration/test_output.log | grep -E "(PASS|FAIL|ok|FAIL)" | tail -5
    
    echo ""
    echo "测试成功率检查："
    PASS_COUNT=$(grep -c "PASS:" test_results/integration/test_output.log || echo "0")
    FAIL_COUNT=$(grep -c "FAIL:" test_results/integration/test_output.log || echo "0")
    echo "通过测试: $PASS_COUNT"
    echo "失败测试: $FAIL_COUNT"
fi

echo ""
echo "============================================="
echo "8. 文件完整性检查"
echo "============================================="

echo "检查测试文件完整性..."
echo "主要测试文件："
ls -la test_results/integration/ 2>/dev/null || echo "测试结果目录不存在"

echo ""
echo "源码文件统计："
echo "核心文件："
ls -la internal/webdav/*_test.go 2>/dev/null | wc -l | xargs echo "测试文件数量:"
echo "源码文件："
ls -la internal/webdav/*.go 2>/dev/null | wc -l | xargs echo "源码文件数量:"

echo ""
echo "============================================="
echo "集成测试完成！"
echo "============================================="

echo "生成的报告文件："
echo "  📊 覆盖率报告: test_results/integration/coverage.html"
echo "  📈 基准测试: test_results/benchmark_results.txt"
echo "  📋 测试日志: test_results/integration/test_output.log"
echo "  📄 覆盖率摘要: test_results/unit_coverage_summary.txt"

if [ -f test_results/integration/coverage.html ]; then
    echo ""
    echo "💡 要查看覆盖率报告，请在浏览器中打开："
    echo "   file://$(pwd)/test_results/integration/coverage.html"
fi

echo ""
echo "============================================="
echo "性能目标验证："
echo "============================================="

# 检查是否生成了基准测试报告
if [ -f test_results/benchmark_results.txt ]; then
    echo "基准性能测试已执行"
    echo "请查看 test_results/benchmark_results.txt 获取详细性能指标"
else
    echo "⚠️  基准测试报告未生成，可能需要手动执行"
    echo "   运行命令: go test -bench=BenchmarkProppatch -benchmem ./internal/webdav/"
fi

echo ""
echo "🎯 集成测试执行完成！"