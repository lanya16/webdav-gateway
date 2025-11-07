# WebDAV Gateway 快速开始

本快速开始指南将帮助您快速部署和使用WebDAV Gateway。

## 系统要求

- Go 1.21+
- SQLite3
- 2GB可用内存
- Linux/macOS/Windows

## 安装步骤

### 1. 克隆仓库
```bash
git clone https://github.com/lanya16/webdav-gateway.git
cd webdav-gateway
```

### 2. 编译
```bash
# 使用Makefile
make build

# 或手动编译
go build -o webdav-gateway cmd/server/main.go
```

### 3. 配置

创建配置文件 `config.yaml`：
```yaml
server:
  port: 8080
  host: "0.0.0.0"
  base_url: "http://localhost:8080"

auth:
  jwt_secret: "your-secret-key"
  token_expiration: 24h

storage:
  provider: "local"
  local:
    path: "./data"

webdav:
  root_path: "/"
  max_file_size: 100MB
```

### 4. 启动服务
```bash
./webdav-gateway --config config.yaml
```

## Docker 部署

### 使用 Docker Compose
```bash
# 启动
docker-compose up -d

# 停止
docker-compose down
```

### 使用 Docker
```bash
# 构建镜像
docker build -t webdav-gateway .

# 运行容器
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config.yaml:/app/config.yaml \
  webdav-gateway
```

## 基本使用

### 启动后访问
- Web界面：http://localhost:8080
- WebDAV服务：http://localhost:8080/webdav/
- API文档：http://localhost:8080/docs

### 挂载WebDAV
```bash
# Linux
mount -t davfs http://localhost:8080/webdav/ /mnt/webdav

# macOS
mount_webdav http://localhost:8080/webdav/ /Volumes/webdav

# Windows
# 使用WebDAV客户端或资源管理器
```

## 性能优化

### 数据库优化
```bash
# 创建索引
sqlite3 data/properties.db "CREATE INDEX idx_resource_id ON properties(resource_id);"
```

### 文件系统优化
```bash
# 增加打开文件限制
ulimit -n 65536
```

## 监控

### 健康检查
```bash
curl http://localhost:8080/health
```

### 日志
```bash
tail -f logs/webdav.log
```

## 故障排除

### 常见问题

1. **端口被占用**
   ```bash
   lsof -i :8080
   kill -9 <PID>
   ```

2. **权限问题**
   ```bash
   chown -R webdav:webdav data/
   chmod 755 data/
   ```

3. **内存不足**
   ```bash
   # 减少缓存大小
   # 调整worker数量
   ```

## 更多信息

详细配置选项请参考 [DEPLOYMENT.md](DEPLOYMENT.md)
API文档请参考 [API.md](API.md)