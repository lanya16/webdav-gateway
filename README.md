# WebDAV网关系统

基于Go语言的高性能WebDAV网关系统，支持Windows网络驱动器映射、文件管理、用户认证和文件分享功能。

## 功能特性

### 核心功能
- 完整的WebDAV协议支持（PROPFIND, GET, PUT, DELETE, MKCOL, MOVE, COPY, LOCK, UNLOCK等）
- Windows网络驱动器映射支持
- MinIO S3存储集成
- JWT Token认证
- 用户存储隔离
- 大文件上传下载
- 文件分享功能（密码保护、过期时间、下载次数限制）
- 存储配额管理
- 🔒 **WebDAV锁定机制**（支持EXCLUSIVE/SHARED锁，持久化存储，冲突检测）

### 技术特点
- 高性能Go语言实现
- S3兼容存储后端
- PostgreSQL数据持久化
- Redis缓存支持
- Docker容器化部署
- RESTful API

## 技术栈

- **后端框架**: Gin (Go)
- **数据库**: PostgreSQL 15
- **缓存**: Redis 7
- **对象存储**: MinIO (S3兼容)
- **认证**: JWT
- **容器化**: Docker & Docker Compose

## 快速开始

### 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- Go 1.21+ (仅用于本地开发)

### 使用Docker Compose部署（推荐）

1. 克隆项目
```bash
git clone <repository-url>
cd webdav-gateway
```

2. 启动所有服务
```bash
chmod +x scripts/*.sh
./scripts/start.sh
```

这将启动以下服务：
- WebDAV Gateway (端口8080)
- PostgreSQL (端口5432)
- Redis (端口6379)
- MinIO (端口9000, 控制台9001)

3. 验证服务
```bash
curl http://localhost:8080/health
```

### 本地开发

1. 启动依赖服务
```bash
cd deployments/docker
docker-compose up -d postgres redis minio
```

2. 配置环境变量
```bash
cp .env.example .env
# 编辑.env文件，配置数据库等连接信息
```

3. 运行应用
```bash
go run cmd/server/main.go cmd/server/auth_handlers.go cmd/server/share_handlers.go
```

## API使用指南

### 1. 用户注册

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123",
    "display_name": "Test User"
  }'
```

### 2. 用户登录

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

返回示例：
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "username": "testuser",
    "email": "test@example.com",
    "display_name": "Test User",
    "storage_quota": 10737418240,
    "storage_used": 0
  }
}
```

### 3. 获取用户信息

```bash
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer <your-token>"
```

### 4. WebDAV操作

#### 上传文件
```bash
curl -X PUT http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: text/plain" \
  -d "Hello WebDAV!"
```

#### 下载文件
```bash
curl -X GET http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <your-token>"
```

#### 列出目录
```bash
curl -X PROPFIND http://localhost:8080/webdav/ \
  -H "Authorization: Bearer <your-token>" \
  -H "Depth: 1"
```

#### 创建目录
```bash
curl -X MKCOL http://localhost:8080/webdav/newfolder \
  -H "Authorization: Bearer <your-token>"
```

#### 删除文件
```bash
curl -X DELETE http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <your-token>"
```

#### 创建文件锁定（EXCLUSIVE锁）
```bash
curl -X LOCK http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/xml" \
  -H "Depth: 0" \
  -H "Timeout: Second-3600" \
  -d '<?xml version="1.0"?>
<lockinfo xmlns="DAV:">
  <lockscope><exclusive/></lockscope>
  <locktype><write/></locktype>
  <owner><href>mailto:user@example.com</href></owner>
</lockinfo>'
```

#### 创建文件锁定（SHARED锁）
```bash
curl -X LOCK http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/xml" \
  -H "Depth: 0" \
  -H "Timeout: Second-3600" \
  -d '<?xml version="1.0"?>
<lockinfo xmlns="DAV:">
  <lockscope><shared/></lockscope>
  <locktype><write/></locktype>
  <owner><href>mailto:user@example.com</href></owner>
</lockinfo>'
```

#### 解除文件锁定
```bash
curl -X UNLOCK http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <your-token>" \
  -H "Lock-Token: <opaquelocktoken:550e8400-e29b-41d4-a716-446655440000>"
```

#### 查询文件锁定状态
```bash
curl -X PROPFIND http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <your-token>" \
  -H "Depth: 0" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?>
<propfind xmlns="DAV:">
  <prop>
    <lockdiscovery/>
    <supportedlock/>
  </prop>
</propfind>'
```

### 5. 文件分享

#### 创建分享链接
```bash
curl -X POST http://localhost:8080/api/shares \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "file_path": "/test.txt",
    "share_name": "测试文件分享",
    "password": "share123",
    "expires_in": 168,
    "max_downloads": 10,
    "permissions": "read"
  }'
```

返回示例：
```json
{
  "share_url": "http://localhost:8080/share/abc123...",
  "share_token": "abc123...",
  "expires_at": "2024-01-15T10:00:00Z"
}
```

#### 访问分享
```bash
curl -X POST http://localhost:8080/share/<token>/access \
  -H "Content-Type: application/json" \
  -d '{
    "password": "share123"
  }'
```

#### 列出我的分享
```bash
curl -X GET http://localhost:8080/api/shares \
  -H "Authorization: Bearer <your-token>"
```

#### 删除分享
```bash
curl -X DELETE http://localhost:8080/api/shares/<share-id> \
  -H "Authorization: Bearer <your-token>"
```

## Windows网络驱动器映射

### 方法1: 使用资源管理器

1. 打开"此电脑"
2. 点击"映射网络驱动器"
3. 输入地址：`http://localhost:8080/webdav`
4. 选择"使用其他凭据连接"
5. 输入JWT Token作为密码（用户名可以留空）

### 方法2: 使用命令行

```cmd
net use Z: http://localhost:8080/webdav /user:token <your-jwt-token>
```

注意：Windows WebDAV客户端可能需要特殊配置才能支持JWT认证。建议使用专业的WebDAV客户端如：
- Cyberduck
- WinSCP
- Mountain Duck
- WebDrive

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| SERVER_HOST | 服务器监听地址 | 0.0.0.0 |
| SERVER_PORT | 服务器端口 | 8080 |
| DB_HOST | PostgreSQL主机 | localhost |
| DB_PORT | PostgreSQL端口 | 5432 |
| DB_USER | 数据库用户 | webdav |
| DB_PASSWORD | 数据库密码 | webdav_password |
| REDIS_HOST | Redis主机 | localhost |
| REDIS_PORT | Redis端口 | 6379 |
| MINIO_ENDPOINT | MinIO地址 | localhost:9000 |
| MINIO_ACCESS_KEY | MinIO访问密钥 | minioadmin |
| MINIO_SECRET_KEY | MinIO秘密密钥 | minioadmin |
| JWT_SECRET | JWT签名密钥 | 需要修改 |
| DEFAULT_STORAGE_QUOTA | 默认存储配额(字节) | 10737418240 (10GB) |
| ENABLE_LOCKING | 启用锁定功能 | true |
| LOCK_TIMEOUT_DEFAULT | 默认锁定超时(秒) | 3600 |
| LOCK_PERSISTENCE_ENABLED | 启用锁定持久化 | false |
| LOCK_PERSISTENCE_DRIVER | 持久化驱动 | memory |

完整配置请参考 `.env.example` 文件。

## 项目结构

```
webdav-gateway/
├── cmd/
│   └── server/              # 主服务入口
├── internal/
│   ├── auth/                # 认证模块
│   ├── webdav/              # WebDAV协议处理
│   ├── storage/             # S3存储适配器
│   ├── share/               # 文件分享
│   ├── middleware/          # 中间件
│   ├── config/              # 配置管理
│   └── models/              # 数据模型
├── deployments/
│   └── docker/              # Docker部署文件
├── scripts/                 # 部署脚本
├── docs/                    # 文档
├── go.mod                   # Go模块定义
└── README.md
```

## 数据库Schema

### users表
存储用户信息，包括认证信息和存储配额。

### file_shares表
存储文件分享记录，包括分享链接、密码、过期时间等。

详细schema请查看 `deployments/docker/schema.sql`

## 安全建议

1. 修改默认的JWT_SECRET
2. 使用强密码
3. 启用HTTPS（生产环境）
4. 定期更新依赖
5. 配置防火墙规则
6. 限制API访问频率
7. 备份数据库

## 监控和日志

### 查看应用日志
```bash
cd deployments/docker
docker-compose logs -f webdav-gateway
```

### 查看所有服务日志
```bash
docker-compose logs -f
```

### 健康检查
```bash
curl http://localhost:8080/health
```

## 故障排除

### 问题1: 无法连接到数据库
- 检查PostgreSQL是否正常运行：`docker-compose ps`
- 检查数据库连接配置
- 查看数据库日志：`docker-compose logs postgres`

### 问题2: MinIO存储错误
- 检查MinIO是否正常运行
- 验证访问密钥配置
- 访问MinIO控制台：http://localhost:9001

### 问题3: WebDAV映射失败
- 确认JWT Token有效
- 检查网络连接
- 尝试使用专业WebDAV客户端
- 查看应用日志排查错误

## 性能优化

1. 调整数据库连接池大小
2. 配置Redis缓存策略
3. 使用CDN加速文件下载
4. 启用MinIO分布式部署
5. 配置负载均衡

## 开发指南

### 添加新功能

1. 在 `internal/` 相应模块中添加业务逻辑
2. 在 `cmd/server/` 中添加HTTP处理器
3. 更新路由配置
4. 编写单元测试
5. 更新文档

### 运行测试
```bash
go test ./...
```

### 代码格式化
```bash
go fmt ./...
```

## License

MIT License

## 贡献

欢迎提交Issue和Pull Request！

## 联系方式

- 项目主页：<repository-url>
- 问题反馈：<issues-url>