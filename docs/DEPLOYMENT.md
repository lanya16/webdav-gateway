# WebDAV Gateway 部署指南

## 概述

本指南提供WebDAV Gateway的生产环境部署最佳实践。

## 系统要求

### 最低要求
- CPU: 2核心
- 内存: 4GB RAM
- 存储: 50GB SSD
- 网络: 100Mbps

### 推荐配置
- CPU: 4核心
- 内存: 8GB RAM
- 存储: 200GB SSD
- 网络: 1Gbps

### 支持的操作系统
- Linux (Ubuntu 20.04+, CentOS 8+, RHEL 8+)
- macOS (10.15+)
- Windows Server 2019+

## 部署方式

### 1. Docker Compose 部署（推荐）

```bash
# 克隆仓库
git clone https://github.com/lanya16/webdav-gateway.git
cd webdav-gateway

# 使用生产环境配置
cp deployments/docker/docker-compose.prod.yml deployments/docker/docker-compose.yml

# 创建环境变量文件
cat > .env << EOF
# WebDAV Gateway 配置
WEBDAV_JWT_SECRET=your-super-secret-jwt-key
WEBDAV_TOKEN_EXPIRATION=24h
WEBDAV_MAX_FILE_SIZE=100MB
WEBDAV_MAX_UPLOAD_SIZE=500MB

# PostgreSQL 配置
POSTGRES_DB=webdav
POSTGRES_USER=webdav
POSTGRES_PASSWORD=webdav123

# Redis 配置
REDIS_PASSWORD=redis123

# MinIO 配置
MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=minio123
EOF

# 启动服务
cd deployments/docker
docker-compose up -d

# 查看服务状态
docker-compose ps
```

### 2. Kubernetes 部署

```yaml
# webdav-gateway.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webdav-gateway
  namespace: webdav
spec:
  replicas: 3
  selector:
    matchLabels:
      app: webdav-gateway
  template:
    metadata:
      labels:
        app: webdav-gateway
    spec:
      containers:
      - name: webdav-gateway
        image: lanya16/webdav-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: WEBDAV_JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: webdav-secrets
              key: jwt-secret
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: webdav-secrets
              key: database-url
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: webdav-gateway-service
  namespace: webdav
spec:
  selector:
    app: webdav-gateway
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

```bash
# 创建命名空间
kubectl create namespace webdav

# 创建密钥
kubectl create secret generic webdav-secrets \
  --from-literal=jwt-secret="your-jwt-secret" \
  --from-literal=database-url="postgresql://user:pass@host:5432/db" \
  --namespace=webdav

# 部署
kubectl apply -f webdav-gateway.yaml

# 检查部署状态
kubectl get pods -n webdav
kubectl get services -n webdav
```

### 3. 裸机部署

```bash
# 安装依赖
sudo apt update
sudo apt install -y golang-go sqlite3 postgresql-client redis-tools nginx

# 编译二进制文件
make build
sudo cp webdav-gateway /usr/local/bin/
sudo chmod +x /usr/local/bin/webdav-gateway

# 创建用户和服务
sudo useradd -r -s /bin/false webdav
sudo mkdir -p /etc/webdav-gateway /var/lib/webdav-gateway
sudo chown webdav:webdav /var/lib/webdav-gateway

# 创建服务文件
sudo tee /etc/systemd/system/webdav-gateway.service << EOF
[Unit]
Description=WebDAV Gateway
After=network.target

[Service]
Type=simple
User=webdav
Group=webdav
WorkingDirectory=/var/lib/webdav-gateway
ExecStart=/usr/local/bin/webdav-gateway --config /etc/webdav-gateway/config.yaml
Restart=always
RestartSec=5
Environment=WEBDAV_JWT_SECRET=your-jwt-secret

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable webdav-gateway
sudo systemctl start webdav-gateway

# 配置Nginx反向代理
sudo tee /etc/nginx/sites-available/webdav << EOF
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/webdav /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## 配置管理

### 环境变量

| 变量名 | 描述 | 默认值 | 生产环境建议 |
|--------|------|--------|-------------|
| `WEBDAV_JWT_SECRET` | JWT签名密钥 | 无 | 强随机字符串（32字符+） |
| `WEBDAV_TOKEN_EXPIRATION` | Token过期时间 | 24h | 根据需求调整 |
| `WEBDAV_MAX_FILE_SIZE` | 最大文件大小 | 100MB | 500MB-1GB |
| `WEBDAV_MAX_UPLOAD_SIZE` | 最大上传大小 | 100MB | 500MB-2GB |
| `DATABASE_URL` | 数据库连接字符串 | 无 | PostgreSQL（推荐） |
| `REDIS_URL` | Redis连接字符串 | 无 | 高可用配置 |

### 配置文件

```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"
  base_url: "https://your-domain.com"
  enable_cors: true
  max_request_size: 1048576 # 1MB

auth:
  jwt_secret: "${WEBDAV_JWT_SECRET}"
  token_expiration: "24h"
  refresh_token_expiration: "168h" # 7天
  max_login_attempts: 5
  lockout_duration: "15m"

storage:
  provider: "postgresql" # postgresql, sqlite, s3
  postgresql:
    url: "${DATABASE_URL}"
    max_idle_connections: 10
    max_open_connections: 100
    max_lifetime: "1h"
  s3:
    endpoint: "https://s3.amazonaws.com"
    access_key_id: "${S3_ACCESS_KEY}"
    secret_access_key: "${S3_SECRET_KEY}"
    bucket: "webdav-gateway"

webdav:
  root_path: "/"
  max_file_size: 104857600 # 100MB
  max_upload_size: 104857600 # 100MB
  enable_locking: true
  lock_timeout: 3600 # 1小时
  enable_etag: true
  enable_compression: true

logging:
  level: "info"
  format: "json"
  output: "stdout"
  file: "/var/log/webdav-gateway.log"
  max_size: "100MB"
  max_backups: 5
  max_age: 28

monitoring:
  enable_prometheus: true
  prometheus_port: 9090
  enable_tracing: true
  trace_sampling_rate: 0.1
```

## 锁定持久化配置

### PostgreSQL 配置

```sql
-- 创建锁定表
CREATE TABLE IF NOT EXISTS locks (
    token VARCHAR(255) PRIMARY KEY,
    resource_path TEXT NOT NULL,
    user_id UUID REFERENCES users(id),
    lock_type VARCHAR(20) NOT NULL CHECK (lock_type IN ('exclusive', 'shared')),
    depth INTEGER NOT NULL DEFAULT 0,
    owner TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_locks_resource_path ON locks(resource_path);
CREATE INDEX IF NOT EXISTS idx_locks_user_id ON locks(user_id);
CREATE INDEX IF NOT EXISTS idx_locks_expires_at ON locks(expires_at);

-- 创建清理过期锁定的函数
CREATE OR REPLACE FUNCTION cleanup_expired_locks()
RETURNS void AS $$
BEGIN
    DELETE FROM locks WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- 创建定期清理任务
SELECT cron.schedule('cleanup-locks', '*/5 * * * *', 'SELECT cleanup_expired_locks();');
```

### Redis 配置

```yaml
# Redis 锁定配置
redis:
  url: "${REDIS_URL}"
  key_prefix: "webdav:lock:"
  ttl: 3600 # 1小时
  enable_lua_scripting: true
```

## 监控配置

### Prometheus 配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'webdav-gateway'
    static_configs:
      - targets: ['webdav-gateway:9090']
    scrape_interval: 10s
    metrics_path: /metrics
```

### Grafana 仪表板

```json
{
  "dashboard": {
    "title": "WebDAV Gateway",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(webdav_requests_total[5m])"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph", 
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(webdav_request_duration_seconds_bucket[5m]))"
          }
        ]
      }
    ]
  }
}
```

## 安全配置

### SSL/TLS 配置

```yaml
# 使用 Let's Encrypt
sudo apt install certbot
sudo certbot --nginx -d your-domain.com

# 或手动配置
nginx:
  ssl:
    certificate: "/path/to/cert.pem"
    certificate_key: "/path/to/key.pem"
    protocols: "TLSv1.2 TLSv1.3"
    ciphers: "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384"
```

### 防火墙配置

```bash
# Ubuntu/Debian
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable

# CentOS/RHEL
sudo firewall-cmd --permanent --add-service=ssh
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

## 性能优化

### 数据库优化

```sql
-- PostgreSQL 优化
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET work_mem = '4MB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
SELECT pg_reload_conf();
```

### WebDAV Gateway 优化

```yaml
# 性能配置
server:
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  max_header_bytes: 1048576

webdav:
  enable_compression: true
  compression_level: 6
  enable_etag: true
  enable_conditional_requests: true

# 连接池配置
database:
  max_idle_connections: 20
  max_open_connections: 100
  max_lifetime: "2h"
  connection_timeout: "10s"

redis:
  max_idle_connections: 20
  max_active_connections: 100
  idle_timeout: "5m"
  wait: true
```

## 备份策略

### 数据库备份

```bash
#!/bin/bash
# backup-db.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/webdav-gateway"
DATABASE_URL="postgresql://user:pass@host:5432/db"

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份数据库
pg_dump $DATABASE_URL > $BACKUP_DIR/db_$DATE.sql

# 压缩备份文件
gzip $BACKUP_DIR/db_$DATE.sql

# 清理7天前的备份
find $BACKUP_DIR -name "db_*.sql.gz" -mtime +7 -delete

# 可选：上传到云存储
# aws s3 cp $BACKUP_DIR/db_$DATE.sql.gz s3://your-backup-bucket/webdav/
```

### 文件存储备份

```bash
#!/bin/bash
# backup-storage.sh

DATE=$(date +%Y%m%d_%H%M%S)
SOURCE_DIR="/var/lib/webdav-gateway/storage"
BACKUP_DIR="/backup/webdav-gateway/storage"

# 创建tar归档
tar -czf $BACKUP_DIR/storage_$DATE.tar.gz -C $SOURCE_DIR .

# 清理30天前的备份
find $BACKUP_DIR -name "storage_*.tar.gz" -mtime +30 -delete
```

## 故障排除

### 常见问题

1. **服务启动失败**
   ```bash
   # 检查日志
   docker-compose logs webdav-gateway
   # 或
   journalctl -u webdav-gateway -f
   
   # 检查配置文件
   /usr/local/bin/webdav-gateway --validate-config /etc/webdav-gateway/config.yaml
   ```

2. **数据库连接失败**
   ```bash
   # 测试数据库连接
   psql $DATABASE_URL -c "SELECT 1;"
   
   # 检查数据库状态
   sudo systemctl status postgresql
   ```

3. **锁定功能不工作**
   ```bash
   # 检查锁定表
   psql $DATABASE_URL -c "SELECT * FROM locks;"
   
   # 检查Redis连接
   redis-cli ping
   ```

4. **性能问题**
   ```bash
   # 查看资源使用
   top
   htop
   docker stats
   
   # 查看慢查询
   psql $DATABASE_URL -c "SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
   ```

### 日志分析

```bash
# 实时查看日志
tail -f /var/log/webdav-gateway.log | jq '.'

# 错误统计
grep "error" /var/log/webdav-gateway.log | jq '.level, .message' | sort | uniq -c

# 请求统计
grep "webdav_request" /var/log/webdav-gateway.log | jq '.path, .method, .status' | sort | uniq -c
```

## 高可用配置

### 负载均衡

```yaml
# nginx.conf upstream 配置
upstream webdav_backend {
    least_conn;
    server 10.0.1.10:8080 weight=1 max_fails=3 fail_timeout=30s;
    server 10.0.1.11:8080 weight=1 max_fails=3 fail_timeout=30s;
    server 10.0.1.12:8080 weight=1 max_fails=3 fail_timeout=30s;
    
    keepalive 32;
}

server {
    location / {
        proxy_pass http://webdav_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 保持连接
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }
}
```

### 数据库高可用

```yaml
# PostgreSQL 主从配置
postgresql:
  primary:
    host: postgres-master
    port: 5432
  replicas:
    - host: postgres-replica-1
      port: 5432
    - host: postgres-replica-2
      port: 5432
  connection_pooling: "pgbouncer"
```

## 更新和升级

### 滚动更新

```bash
# Docker Compose
docker-compose pull
docker-compose up -d

# Kubernetes
kubectl set image deployment/webdav-gateway webdav-gateway=lanya16/webdav-gateway:v2.0.0
kubectl rollout status deployment/webdav-gateway
```

### 数据库迁移

```bash
# 运行迁移脚本
kubectl exec -it webdav-gateway-xxx -- /usr/local/bin/webdav-gateway migrate

# 或手动执行
psql $DATABASE_URL -f migrations/upgrade.sql
```

## 监控告警

### 关键指标

- 服务可用性 < 99.9%
- 平均响应时间 > 2秒
- 错误率 > 1%
- CPU使用率 > 80%
- 内存使用率 > 85%
- 磁盘使用率 > 90%

### 告警规则

```yaml
# alertmanager.yml
groups:
  - name: webdav-gateway
    rules:
      - alert: HighErrorRate
        expr: rate(webdav_requests_total{status=~"5.."}[5m]) > 0.01
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: High error rate detected
          
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(webdav_request_duration_seconds_bucket[5m])) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: High response time detected
```

## 维护窗口

### 计划维护

1. **通知用户** - 提前48小时通知
2. **备份数据** - 完整备份
3. **执行维护** - 低峰时段进行
4. **验证功能** - 测试所有关键功能
5. **通知完成** - 向用户报告结果

### 紧急维护

1. **立即备份** - 保护关键数据
2. **快速修复** - 优先恢复服务
3. **事后分析** - 分析根本原因
4. **预防措施** - 防止再次发生

通过遵循本部署指南，您可以成功部署和运营WebDAV Gateway的生产环境。