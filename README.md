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

## 编译错误修复说明

此版本已修复所有编译错误：

- ✅ PropertyError类型导入问题已解决
- ✅ 重复导入问题已修复
- ✅ Property和DatabaseProperty类型协调
- ✅ SQL构建器Args()方法已添加
- ✅ 时间戳处理优化
- ✅ XML验证和解析功能完整

## License

MIT License

## 贡献

欢迎提交Issue和Pull Request！