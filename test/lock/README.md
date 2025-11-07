# WebDAV LOCK/UNLOCK测试工具包

本测试工具包提供了完整的WebDAV LOCK/UNLOCK机制测试解决方案，包括单元测试、集成测试、性能测试和客户端兼容性测试。

## 📁 文件结构

```
test/lock/
├── lock_test.go                    # Go单元测试文件 - 基础锁定功能测试
├── persistence_test.go            # Go单元测试文件 - 持久化功能测试
├── integration_test.go            # Go单元测试文件 - 集成场景测试
├── test-lock-integration.sh       # 集成测试脚本
├── test-lock-performance.sh       # 性能测试脚本
├── test-data-generator.go         # 测试数据生成工具
├── client-compatibility.md        # 客户端兼容性测试指南
├── test-report-template.md        # 测试报告模板
└── README.md                      # 本文件
```

## 🚀 快速开始

### 1. 环境准备

确保您的系统已安装以下工具：
- Go 1.16+
- curl
- bc (用于性能计算)
- 基础Unix工具 (bash, grep, sed等)

### 2. 运行单元测试

```bash
cd /path/to/webdav-gateway

# 运行所有锁定相关测试
go test ./test/lock/... -v

# 运行特定测试文件
go test ./test/lock/lock_test.go -v
go test ./test/lock/persistence_test.go -v
go test ./test/lock/integration_test.go -v

# 运行基准测试
go test ./test/lock/... -bench=.

# 生成测试覆盖率报告
go test ./test/lock/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### 3. 编译测试数据生成工具

```bash
cd /path/to/webdav-gateway/test/lock

# 编译测试数据生成工具
go build -o test-data-generator test-data-generator.go

# 生成测试数据
./test-data-generator \
  --url "http://localhost:8080" \
  --user "testuser" \
  --pass "testpass" \
  --output "./generated-test-data" \
  --tokens 100 \
  --format "xml,json,curl,loadtest"
```

### 4. 运行集成测试

```bash
cd /path/to/webdav-gateway/test/lock

# 设置环境变量
export WEBDAV_BASE_URL="http://localhost:8080"
export TEST_USER="testuser"
export TEST_PASSWORD="testpass"

# 运行集成测试
bash test-lock-integration.sh

# 使用自定义参数运行
WEBDAV_BASE_URL="http://your-server.com" \
TEST_USER="admin" \
TEST_PASSWORD="secret" \
bash test-lock-integration.sh
```

### 5. 运行性能测试

```bash
cd /path/to/webdav-gateway/test/lock

# 运行性能测试
bash test-lock-performance.sh

# 快速测试模式（减少测试量）
bash test-lock-performance.sh --quick

# 使用自定义参数
bash test-lock-performance.sh \
  --url "http://localhost:8080" \
  --user "testuser" \
  --password "testpass" \
  --concurrent 20 \
  --operations 100 \
  --files 200
```

## 📊 测试覆盖范围

### 基础锁定测试
- ✅ EXCLUSIVE锁定创建和管理
- ✅ SHARED锁定创建和管理
- ✅ 锁定令牌验证
- ✅ 锁定释放操作
- ✅ 锁定状态查询

### 锁定冲突测试
- ✅ EXCLUSIVE vs EXCLUSIVE冲突
- ✅ EXCLUSIVE vs SHARED冲突
- ✅ SHARED vs SHARED兼容性
- ✅ 父子目录锁定冲突
- ✅ 深度锁定冲突处理

### 锁定超时测试
- ✅ 锁定自动过期机制
- ✅ 超时后访问控制
- ✅ 锁定清理机制
- ✅ 锁定刷新防止过期

### 客户端兼容性测试
- ✅ Windows WebDAV客户端
- ✅ macOS Finder兼容性
- ✅ Linux davfs2测试
- ✅ Cyberduck客户端
- ✅ FileZilla客户端
- ✅ 命令行工具兼容性

### 边界条件测试
- ✅ 空路径和特殊字符路径
- ✅ 超长路径处理
- ✅ 并发锁定操作
- ✅ 大量锁定管理

### 错误处理测试
- ✅ 无效XML格式处理
- ✅ 缺失必需元素处理
- ✅ 无效锁定类型处理
- ✅ 锁定令牌格式错误
- ✅ 网络中断恢复

### 性能测试
- ✅ 锁定创建性能
- ✅ 锁定查找性能
- ✅ 锁定清理性能
- ✅ 并发锁定性能
- ✅ 内存使用效率
- ✅ 锁定超时处理性能

## 🛠️ 详细使用说明

### 测试数据生成工具

测试数据生成工具可以生成各种格式的测试数据：

```bash
# 基本用法
./test-data-generator

# 指定输出目录和格式
./test-data-generator \
  --output "/tmp/webdav-test-data" \
  --format "xml,json,curl"

# 指定WebDAV服务器和认证
./test-data-generator \
  --url "http://webdav.example.com" \
  --user "admin" \
  --pass "secret" \
  --tokens 200

# 生成性能测试数据
./test-data-generator \
  --perf-ops 1000 \
  --format "perf"
```

### 集成测试脚本

集成测试脚本提供完整的端到端测试：

```bash
# 查看帮助
bash test-lock-integration.sh --help

# 运行所有测试
bash test-lock-integration.sh

# 自定义测试参数
WEBDAV_BASE_URL="https://secure-webdav.example.com" \
TEST_USER="username" \
TEST_PASSWORD="password" \
bash test-lock-integration.sh
```

### 性能测试脚本

性能测试脚本提供详细的性能指标：

```bash
# 查看所有选项
bash test-lock-performance.sh --help

# 标准性能测试
bash test-lock-performance.sh

# 快速测试（适合CI/CD）
bash test-lock-performance.sh --quick

# 高负载测试
bash test-lock-performance.sh \
  --concurrent 50 \
  --operations 200 \
  --files 500
```

## 📈 测试报告

### 自动生成的报告

运行测试后会自动生成以下报告：

1. **测试日志文件**
   - `test-lock-integration.log` - 集成测试日志
   - `test-lock-performance.log` - 性能测试日志

2. **测试报告**
   - `test-lock-integration-report.md` - 集成测试报告
   - `test-lock-performance-report.md` - 性能测试报告

3. **性能数据**
   - `performance-results.json` - JSON格式的性能结果

### 使用报告模板

使用提供的报告模板生成自定义报告：

```bash
# 复制模板
cp test-report-template.md my-test-report.md

# 编辑模板，替换占位符
# {{VERSION}} -> "1.0.0"
# {{DATE}} -> "2025-01-01"
# {{TESTER}} -> "测试工程师"
# ... 其他占位符
```

## 🔧 高级配置

### 自定义测试环境

创建自定义测试配置文件：

```bash
# 创建配置文件
cat > test-config.env << EOF
WEBDAV_BASE_URL="http://your-server.com"
TEST_USER="your-username"
TEST_PASSWORD="your-password"
TEST_DIR="/your-test-directory"
CONCURRENT_USERS=10
OPERATIONS_PER_USER=50
TIMEOUT_SECONDS=30
EOF

# 加载配置
source test-config.env

# 运行测试
bash test-lock-integration.sh
bash test-lock-performance.sh
```

### 持续集成集成

在CI/CD管道中使用：

```yaml
# .github/workflows/webdav-lock-test.yml
name: WebDAV LOCK/UNLOCK Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    
    - name: Setup Go
      uses: actions/setup-go@v2
      with:
        go-version: 1.19
    
    - name: Run Unit Tests
      run: |
        cd webdav-gateway
        go test ./test/lock/... -v -coverprofile=coverage.out
    
    - name: Run Integration Tests
      env:
        WEBDAV_BASE_URL: ${{ secrets.WEBDAV_URL }}
        TEST_USER: ${{ secrets.TEST_USER }}
        TEST_PASSWORD: ${{ secrets.TEST_PASSWORD }}
      run: |
        cd webdav-gateway/test/lock
        bash test-lock-integration.sh
    
    - name: Run Performance Tests
      env:
        WEBDAV_BASE_URL: ${{ secrets.WEBDAV_URL }}
        TEST_USER: ${{ secrets.TEST_USER }}
        TEST_PASSWORD: ${{ secrets.TEST_PASSWORD }}
      run: |
        cd webdav-gateway/test/lock
        bash test-lock-performance.sh --quick
    
    - name: Upload Coverage
      uses: codecov/codecov-action@v2
      with:
        file: ./webdav-gateway/coverage.out
```

### Docker化测试

创建测试容器：

```dockerfile
# Dockerfile.test
FROM golang:1.19

RUN apt-get update && apt-get install -y \
    curl \
    bc \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .

RUN go build ./test/lock/test-data-generator.go -o test-data-generator

CMD ["bash", "./test/lock/test-lock-integration.sh"]
```

```bash
# 构建和运行测试容器
docker build -f Dockerfile.test -t webdav-lock-test .
docker run -e WEBDAV_BASE_URL="http://host.docker.internal:8080" \
           -e TEST_USER="testuser" \
           -e TEST_PASSWORD="testpass" \
           webdav-lock-test
```

## 🐛 故障排除

### 常见问题

1. **测试失败：连接被拒绝**
   ```bash
   # 检查WebDAV服务器是否运行
   curl -I http://localhost:8080
   
   # 检查防火墙设置
   sudo ufw status
   ```

2. **性能测试失败：bc命令未找到**
   ```bash
   # Ubuntu/Debian
   sudo apt-get install bc
   
   # CentOS/RHEL
   sudo yum install bc
   ```

3. **权限错误：无法执行脚本**
   ```bash
   # 添加执行权限
   chmod +x test-lock-integration.sh
   chmod +x test-lock-performance.sh
   ```

4. **测试数据生成失败：XML格式错误**
   ```bash
   # 验证生成的XML格式
   xmllint --noout webdav-lock-test-data.xml
   
   # 检查JSON格式
   jq . webdav-lock-test-data.json
   ```

### 调试模式

启用详细调试：

```bash
# 启用bash调试
set -x
bash test-lock-integration.sh

# 启用Go测试详细输出
go test ./test/lock/... -v -race

# 启用网络调试
curl -v -X LOCK http://localhost:8080/test.txt
```

### 日志分析

分析测试日志：

```bash
# 查看错误日志
grep -i error test-lock-integration.log

# 查看性能数据
grep -i "ops/sec" test-lock-performance.log

# 查看测试统计
grep -i "success\|fail" test-lock-integration.log
```

## 📚 参考资料

### WebDAV规范
- [RFC 4918 - WebDAV](https://tools.ietf.org/html/rfc4918)
- [RFC 2518 - HTTP Extensions for Web Distributed Authoring and Versioning (WebDAV)](https://tools.ietf.org/html/rfc2518)

### 测试相关
- [Go Testing Package](https://golang.org/pkg/testing/)
- [WebDAV客户端兼容性指南](./client-compatibility.md)
- [测试报告模板](./test-report-template.md)

### 相关工具
- [cadaver - 命令行WebDAV客户端](https://www.nottingham.ac.uk/~ppzap4/command.html)
- [Cyberduck - WebDAV客户端](https://cyberduck.io/)
- [davfs2 - Linux WebDAV文件系统](http://savannah.nongnu.org/projects/davfs2)

## 🤝 贡献

欢迎提交问题报告和改进建议！

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 创建 Pull Request

## 📄 许可证

本测试工具包遵循项目主许可证。

---

**注意**: 在生产环境中运行测试前，请确保已备份重要数据，并使用专门的测试环境。