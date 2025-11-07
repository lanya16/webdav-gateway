# WebDAV网关系统 - API文档

## 基础信息

- **Base URL**: `http://localhost:8080`
- **认证方式**: JWT Bearer Token
- **Content-Type**: `application/json`

## 认证相关API

### 1. 用户注册

**请求**

```http
POST /api/auth/register
Content-Type: application/json

{
  "username": "string",      // 必填，3-50字符
  "email": "string",          // 必填，有效邮箱
  "password": "string",       // 必填，至少8字符
  "display_name": "string"    // 可选，最多100字符
}
```

**响应**

```json
{
  "id": "uuid",
  "username": "string",
  "email": "string",
  "display_name": "string",
  "storage_quota": 10737418240,
  "storage_used": 0,
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**状态码**
- 201: 创建成功
- 400: 请求参数错误
- 409: 用户已存在

### 2. 用户登录

**请求**

```http
POST /api/auth/login
Content-Type: application/json

{
  "username": "string",
  "password": "string"
}
```

**响应**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "username": "string",
    "email": "string",
    "display_name": "string",
    "storage_quota": 10737418240,
    "storage_used": 0,
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**状态码**
- 200: 登录成功
- 400: 请求参数错误
- 401: 用户名或密码错误

### 3. 获取当前用户信息

**请求**

```http
GET /api/auth/me
Authorization: Bearer <token>
```

**响应**

```json
{
  "id": "uuid",
  "username": "string",
  "email": "string",
  "display_name": "string",
  "storage_quota": 10737418240,
  "storage_used": 1234567,
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**状态码**
- 200: 成功
- 401: 未授权
- 404: 用户不存在

## WebDAV协议API

所有WebDAV请求都需要Bearer Token认证。

### 1. OPTIONS - 获取支持的方法

**请求**

```http
OPTIONS /webdav/*
Authorization: Bearer <token>
```

**响应头**
- `DAV: 1, 2`
- `Allow: OPTIONS, GET, HEAD, POST, PUT, DELETE, PROPFIND, PROPPATCH, MKCOL, COPY, MOVE, LOCK, UNLOCK`

### 9. LOCK - 创建锁定

**请求**

```http
LOCK /webdav/path/to/resource
Authorization: Bearer <token>
Content-Type: application/xml
Depth: 0|infinity
Timeout: Second-3600|infinite

<?xml version="1.0"?>
<lockinfo xmlns="DAV:">
  <lockscope>
    <exclusive/>  <!-- 或 <shared/> -->
  </lockscope>
  <locktype>
    <write/>
  </locktype>
  <owner>
    <href>mailto:user@example.com</href>  <!-- 或其他标识 -->
  </owner>
</lockinfo>
```

**响应**

```xml
<?xml version="1.0"?>
<D:prop xmlns:D="DAV:">
  <D:lockdiscovery>
    <D:activelock>
      <D:locktype><D:write/></D:locktype>
      <D:lockscope><D:exclusive/></D:lockscope>
      <D:depth>0</D:depth>
      <D:owner>
        <D:href>mailto:user@example.com</D:href>
      </D:owner>
      <D:timeout>Second-3600</D:timeout>
      <D:locktoken>
        <D:href>opaquelocktoken:550e8400-e29b-41d4-a716-446655440000</D:href>
      </D:locktoken>
    </D:activelock>
  </D:lockdiscovery>
  <D:supportedlock>
    <D:lockentry>
      <D:lockscope><D:exclusive/></D:lockscope>
      <D:locktype><D:write/></D:locktype>
    </D:lockentry>
    <D:lockentry>
      <D:lockscope><D:shared/></D:lockscope>
      <D:locktype><D:write/></D:locktype>
    </D:lockentry>
  </D:supportedlock>
</D:prop>
```

**响应头**
- `Content-Type: application/xml; charset=utf-8`
- `Lock-Token: <opaquelocktoken:550e8400-e29b-41d4-a716-446655440000>`

**状态码**
- 200: 创建成功
- 400: 请求参数错误
- 401: 未授权
- 423: 资源已被锁定
- 409: 冲突（父目录不存在等）

**锁定类型说明**
- **EXCLUSIVE**: 排他锁，同一时间仅允许一个客户端持有
- **SHARED**: 共享锁，允许多个客户端同时持有，但与EXCLUSIVE锁互斥

**超时设置**
- `Second-xxx`: 锁定持续xxx秒
- `infinite`: 永久锁定（不推荐）

### 10. UNLOCK - 解除锁定

**请求**

```http
UNLOCK /webdav/path/to/resource
Authorization: Bearer <token>
Lock-Token: <opaquelocktoken:550e8400-e29b-41d4-a716-446655440000>
```

**响应**

```
HTTP/1.1 204 No Content
```

**状态码**
- 204: 解除成功
- 401: 未授权
- 403: 无权限解除该锁定
- 409: Lock-Token无效或不存在

**注意**
- UNLOCK是幂等操作，重复解锁已移除的锁定应返回204
- 只有锁定持有者或具有相应权限的用户才能解除锁定

### 11. 锁定状态检查

通过PROPFIND可以查询资源的锁定状态：

**请求**

```http
PROPFIND /webdav/path/to/resource
Authorization: Bearer <token>
Depth: 0
Content-Type: application/xml

<?xml version="1.0"?>
<propfind xmlns="DAV:">
  <prop>
    <lockdiscovery/>
    <supportedlock/>
  </prop>
</propfind>
```

**响应中的锁定信息**

```xml
<D:propstat>
  <D:prop>
    <D:lockdiscovery>
      <D:activelock>
        <D:locktype><D:write/></D:locktype>
        <D:lockscope><D:exclusive/></D:lockscope>
        <D:depth>0</D:depth>
        <D:owner>
          <D:href>mailto:user@example.com</D:href>
        </D:owner>
        <D:timeout>Second-3600</D:timeout>
        <D:locktoken>
          <D:href>opaquelocktoken:550e8400-e29b-41d4-a716-446655440000</D:href>
        </D:locktoken>
      </D:activelock>
    </D:lockdiscovery>
    <D:supportedlock>
      <D:lockentry>
        <D:lockscope><D:exclusive/></D:lockscope>
        <D:locktype><D:write/></D:locktype>
      </D:lockentry>
      <D:lockentry>
        <D:lockscope><D:shared/></D:lockscope>
        <D:locktype><D:write/></D:locktype>
      </D:lockentry>
    </D:supportedlock>
  </D:prop>
  <D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
```

### 2. PROPFIND - 获取资源属性

**请求**

```http
PROPFIND /webdav/path/to/resource
Authorization: Bearer <token>
Depth: 0|1|infinity
Content-Type: application/xml

<?xml version="1.0"?>
<propfind xmlns="DAV:">
  <prop>
    <displayname/>
    <getcontentlength/>
    <getcontenttype/>
    <getlastmodified/>
    <resourcetype/>
  </prop>
</propfind>
```

**响应**

```xml
<?xml version="1.0"?>
<D:multistatus xmlns:D="DAV:">
  <D:response>
    <D:href>/webdav/test.txt</D:href>
    <D:propstat>
      <D:prop>
        <D:displayname>test.txt</D:displayname>
        <D:getcontentlength>1234</D:getcontentlength>
        <D:getcontenttype>text/plain</D:getcontenttype>
        <D:getlastmodified>Mon, 01 Jan 2024 00:00:00 GMT</D:getlastmodified>
        <D:resourcetype/>
      </D:prop>
      <D:status>HTTP/1.1 200 OK</D:status>
    </D:propstat>
  </D:response>
</D:multistatus>
```

**状态码**
- 207: 多状态响应
- 401: 未授权
- 404: 资源不存在

### 3. GET - 下载文件

**请求**

```http
GET /webdav/path/to/file.txt
Authorization: Bearer <token>
```

**响应**

文件内容（二进制流）

**响应头**
- `Content-Type`: 文件MIME类型
- `Content-Length`: 文件大小
- `Last-Modified`: 最后修改时间
- `ETag`: 实体标签

**状态码**
- 200: 成功
- 401: 未授权
- 404: 文件不存在

### 4. PUT - 上传文件

**请求**

```http
PUT /webdav/path/to/newfile.txt
Authorization: Bearer <token>
Content-Type: text/plain
Content-Length: 1234

[文件内容]
```

**状态码**
- 201: 创建成功
- 204: 更新成功
- 401: 未授权
- 507: 存储空间不足

### 5. DELETE - 删除文件/目录

**请求**

```http
DELETE /webdav/path/to/file.txt
Authorization: Bearer <token>
```

**状态码**
- 204: 删除成功
- 401: 未授权
- 404: 资源不存在

### 6. MKCOL - 创建目录

**请求**

```http
MKCOL /webdav/path/to/newfolder
Authorization: Bearer <token>
```

**状态码**
- 201: 创建成功
- 401: 未授权
- 409: 父目录不存在

### 7. MOVE - 移动文件

**请求**

```http
MOVE /webdav/source/file.txt
Authorization: Bearer <token>
Destination: /webdav/target/file.txt
Overwrite: T|F
```

**状态码**
- 201: 移动成功
- 204: 覆盖成功
- 401: 未授权
- 412: 目标已存在且Overwrite=F

### 8. COPY - 复制文件

**请求**

```http
COPY /webdav/source/file.txt
Authorization: Bearer <token>
Destination: /webdav/target/file.txt
Overwrite: T|F
```

**状态码**
- 201: 复制成功
- 204: 覆盖成功
- 401: 未授权

## 文件分享API

### 1. 创建分享链接

**请求**

```http
POST /api/shares
Authorization: Bearer <token>
Content-Type: application/json

{
  "file_path": "/path/to/file.txt",    // 必填
  "share_name": "分享的文件",           // 可选
  "password": "share123",               // 可选
  "expires_in": 168,                    // 可选，小时数
  "max_downloads": 10,                  // 可选
  "permissions": "read"                 // 可选：read|write
}
```

**响应**

```json
{
  "share_url": "http://localhost:8080/share/abc123...",
  "share_token": "abc123...",
  "expires_at": "2024-01-08T00:00:00Z"
}
```

**状态码**
- 201: 创建成功
- 400: 请求参数错误
- 401: 未授权

### 2. 获取分享信息

**请求**

```http
GET /share/{token}
```

**响应**

```json
{
  "share_name": "分享的文件",
  "file_path": "/path/to/file.txt",
  "expires_at": "2024-01-08T00:00:00Z",
  "download_count": 5,
  "max_downloads": 10,
  "has_password": true
}
```

**状态码**
- 200: 成功
- 404: 分享不存在

### 3. 访问分享（验证密码）

**请求**

```http
POST /share/{token}/access
Content-Type: application/json

{
  "password": "share123"  // 如果设置了密码
}
```

**响应**

```json
{
  "message": "access granted",
  "file_path": "/path/to/file.txt",
  "share_name": "分享的文件"
}
```

**状态码**
- 200: 验证成功
- 401: 密码错误
- 403: 达到下载次数限制
- 404: 分享不存在
- 410: 分享已过期

### 4. 列出我的分享

**请求**

```http
GET /api/shares
Authorization: Bearer <token>
```

**响应**

```json
[
  {
    "id": "uuid",
    "user_id": "uuid",
    "file_path": "/path/to/file.txt",
    "share_token": "abc123...",
    "share_name": "分享的文件",
    "expires_at": "2024-01-08T00:00:00Z",
    "max_downloads": 10,
    "download_count": 5,
    "permissions": "read",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

**状态码**
- 200: 成功
- 401: 未授权

### 5. 删除分享

**请求**

```http
DELETE /api/shares/{id}
Authorization: Bearer <token>
```

**状态码**
- 204: 删除成功
- 401: 未授权
- 404: 分享不存在

## 健康检查API

### 健康状态

**请求**

```http
GET /health
```

**响应**

```json
{
  "status": "healthy",
  "time": 1704067200
}
```

**状态码**
- 200: 服务正常

## 错误响应格式

所有错误响应都遵循以下格式：

```json
{
  "error": "错误描述信息"
}
```

## 通用状态码

- 200: 成功
- 201: 创建成功
- 204: 成功（无内容）
- 207: 多状态（WebDAV）
- 400: 请求参数错误
- 401: 未授权
- 403: 禁止访问
- 404: 资源不存在
- 409: 冲突
- 412: 前置条件失败
- 423: 资源被锁定
- 500: 服务器内部错误
- 507: 存储空间不足

## 锁定相关错误代码

### 423 Locked - 资源被锁定

**响应体**

```xml
<?xml version="1.0" encoding="utf-8"?>
<D:error xmlns:D="DAV:">
  <D:locktoken>
    <D:href>opaquelocktoken:550e8400-e29b-41d4-a716-446655440000</D:href>
  </D:locktoken>
  <D:lockdiscovery>
    <D:activelock>
      <D:locktype><D:write/></D:locktype>
      <D:lockscope><D:exclusive/></D:lockscope>
      <D:depth>0</D:depth>
      <D:owner>
        <D:href>mailto:user@example.com</D:href>
      </D:owner>
      <D:timeout>Second-3600</D:timeout>
      <D:locktoken>
        <D:href>opaquelocktoken:550e8400-e29b-41d4-a716-446655440000</D:href>
      </D:locktoken>
    </D:activelock>
  </D:lockdiscovery>
</D:error>
```

**常见场景**
- 尝试修改被EXCLUSIVE锁定的资源
- 尝试在EXCLUSIVE锁定的目录下创建子资源
- 尝试删除被锁定的资源
- 非锁定持有者尝试解除锁定

**解决建议**
- 等待锁定过期
- 获取锁定持有者的Lock-Token后重新操作
- 联系锁定持有者释放锁定

## 速率限制

当前版本未实施速率限制，生产环境建议配置：

- 登录接口: 5次/分钟/IP
- 注册接口: 3次/小时/IP
- WebDAV上传: 基于存储配额
- API调用: 100次/分钟/用户

## 示例代码

### JavaScript (Fetch API)

```javascript
// 登录
const login = async (username, password) => {
  const response = await fetch('http://localhost:8080/api/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ username, password }),
  });
  return await response.json();
};

// 上传文件
const uploadFile = async (token, filePath, fileContent) => {
  const response = await fetch(`http://localhost:8080/webdav${filePath}`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/octet-stream',
    },
    body: fileContent,
  });
  return response.status === 201;
};

// 创建分享
const createShare = async (token, filePath) => {
  const response = await fetch('http://localhost:8080/api/shares', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      file_path: filePath,
      expires_in: 168,
    }),
  });
  return await response.json();
};
```

### Python (requests)

```python
import requests

# 登录
def login(username, password):
    response = requests.post(
        'http://localhost:8080/api/auth/login',
        json={'username': username, 'password': password}
    )
    return response.json()

# 上传文件
def upload_file(token, file_path, file_content):
    response = requests.put(
        f'http://localhost:8080/webdav{file_path}',
        headers={'Authorization': f'Bearer {token}'},
        data=file_content
    )
    return response.status_code == 201

# 创建分享
def create_share(token, file_path):
    response = requests.post(
        'http://localhost:8080/api/shares',
        headers={'Authorization': f'Bearer {token}'},
        json={'file_path': file_path, 'expires_in': 168}
    )
    return response.json()
```

### cURL

```bash
# 登录
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}'

# 上传文件
curl -X PUT http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: text/plain" \
  -d "Hello WebDAV!"

# 创建分享
curl -X POST http://localhost:8080/api/shares \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"file_path":"/test.txt","expires_in":168}'

# 创建EXCLUSIVE锁
curl -X LOCK http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/xml" \
  -H "Depth: 0" \
  -H "Timeout: Second-3600" \
  -d '<?xml version="1.0"?>
<lockinfo xmlns="DAV:">
  <lockscope><exclusive/></lockscope>
  <locktype><write/></locktype>
  <owner><href>mailto:user@example.com</href></owner>
</lockinfo>'

# 创建SHARED锁
curl -X LOCK http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/xml" \
  -H "Depth: 0" \
  -H "Timeout: Second-3600" \
  -d '<?xml version="1.0"?>
<lockinfo xmlns="DAV:">
  <lockscope><shared/></lockscope>
  <locktype><write/></locktype>
  <owner><href>mailto:user@example.com</href></owner>
</lockinfo>'

# 解除锁定
curl -X UNLOCK http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <token>" \
  -H "Lock-Token: <opaquelocktoken:550e8400-e29b-41d4-a716-446655440000>"

# 查询锁定状态
curl -X PROPFIND http://localhost:8080/webdav/test.txt \
  -H "Authorization: Bearer <token>" \
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