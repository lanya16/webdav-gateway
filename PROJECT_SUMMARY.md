# WebDAV 网关 LOCK/UNLOCK 实现报告蓝图

## 引言与目标

本报告旨在系统化呈现 WebDAV 网关在 LOCK/UNLOCK 能力上的设计与实现现状、关键方法与数据结构、已接入的 HTTP 方法与锁定交互、冲突检测与错误处理机制、响应 XML 的组织与规范对齐情况、测试与验证计划,以及风险与后续迭代方向。目标是在既有代码与功能的基础上,形成一套可执行、可验证、可维护的实现蓝图,确保工程团队能够快速掌握现状、识别差距并推进落地。

本报告围绕以下核心问题展开:
- 当前实现是否满足在 webdav/handler.go 中新增 handleLock/handleUnlock 的目标?
- LOCK 请求体解析、锁定创建与验证、令牌生成、超时与刷新机制是否齐备?
- 冲突检测(EXCLUSIVE/SHARED)与 HTTP 状态码(423/409/400/204)返回是否正确?
- 响应 XML(DAV:lockdiscovery、DAV:locktoken、DAV:lockinfo)是否完整、规范?
- 与现有方法(PROPFIND、GET、HEAD、PUT、DELETE、MKCOL、MOVE、COPY、OPTIONS)的集成是否合理?
- 错误处理与日志是否充分,是否便于调试与运维?
- 测试用例与验证计划如何覆盖功能、规范对齐、性能与回归?
- 后续需要补充的关键功能与优化点是什么?

## 现状与代码结构总览

当前实现已具备较为完整的 WebDAV 锁定基础设施与响应框架:

- 核心结构与管理器:
  - LockManager 提供 CreateLock、GetLock、GetLocksForPath、RemoveLock、CheckLock、CheckExclusiveLock、CheckParentLocks、GetLockDiscovery、CleanExpiredLocks 等方法,能够支持内存中的锁生命周期管理。
  - Lock、ActiveLock、LockInfo、LockScopeInfo、LockTypeInfo、LockToken 等结构体为锁定数据与响应 XML 提供了良好的抽象。
- Handler 集成:
  - Handler 已内嵌 lockManager,并在 PROPFIND 响应中输出 supportedlock 与 lockdiscovery,具备与客户端的基本互操作能力。
  - SendLockedError、CheckExclusiveLock、CheckSharedLock、CheckParentLocks、CheckAnyLock 等方法为冲突检测与错误处理提供了可复用能力。
- HTTP 方法与 OPTIONS:
  - 已实现 GET、HEAD、PUT、DELETE、MKCOL、MOVE、COPY、PROPFIND、OPTIONS 等方法,OPTIONS 的 Allow 头部已包含 LOCK 与 UNLOCK。
- 响应与错误:
  - 423 Locked 错误响应通过 LockedError 结构体实现;在部分读取方法中已调用 CheckSharedLock 以允许共享锁下的读取。

为便于把握能力分布与实现位置,以下表格汇总关键模块与方法。

表 1:关键模块与方法清单
| 模块/文件 | 方法/结构体 | 职责 | 状态 |
|---|---|---|---|
| lock.go | LockManager | 锁创建、查询、移除、冲突检测、过期清理 | 已实现 |
| lock.go | Lock/ActiveLock/LockInfo/LockScopeInfo/LockTypeInfo/LockToken | 锁数据与响应 XML 元素 | 已实现 |
| handler.go | Handler.lockManager | 锁管理器集成 | 已实现 |
| handler.go | SendLockedError/CheckExclusiveLock/CheckSharedLock/CheckParentLocks/CheckAnyLock | 冲突检测与错误处理 | 已实现 |
| handler.go | GET/HEAD/PUT/DELETE/MKCOL/MOVE/COPY/PROPFIND/OPTIONS | HTTP 方法实现与 OPTIONS 头部 | 已实现 |
| handler.go | ResponseProp.SupportedLock/LockDiscovery | PROPFIND 响应属性 | 已实现 |
| handler.go | LockedError | 423 错误响应结构 | 已实现 |
| handler.go | HandleLock/HandleUnlock | LOCK/UNLOCK 方法实现与路由接入 | 已实现(需编译与集成验证) |

## LOCK/UNLOCK 方法设计与实现

LOCK 与 UNLOCK 的实现遵循 RFC 4918 的基本语义,重点在于请求解析、锁创建与验证、令牌生成、超时与刷新、冲突检测与错误处理、标准化响应输出。

- handleLock:
  - 解析请求体中的 XML(lockscope、locktype、owner、timeout、depth)。
  - 验证锁类型与范围(EXCLUSIVE/SHARED),解析超时(Second-xxx 或 infinite),解析深度(0 或 infinity)。
  - 检查资源与父目录锁定冲突;通过 CheckExclusiveLock 与 CheckParentLocks 判断是否返回 423 Locked。
  - 创建锁并回写响应:包含活动锁 discovery 与 supportedlock;响应头回写 Lock-Token。
- handleUnlock:
  - 解析 Lock-Token 头部(去除角括号),验证令牌与资源路径匹配、权限与存在性。
  - 移除锁并返回 204 No Content;在令牌不存在或权限不匹配时返回相应错误。

为明确方法行为与返回码,以下表格给出典型场景。

表 2:方法行为与返回码对照
| 方法 | 场景 | 输入 | 输出 | 状态码 |
|---|---|---|---|---|
| LOCK | 请求体解析失败 | 非法 XML | 错误响应 | 400 |
| LOCK | 锁创建成功 | 合法请求体 | 活动锁 discovery、Lock-Token | 200 |
| LOCK | 资源存在 EXCLUSIVE 锁且非持有者 | 写入类请求 | LockedError | 423 |
| LOCK | 父目录存在 EXCLUSIVE 锁且非持有者 | 写入类请求 | LockedError | 423 |
| UNLOCK | 成功解锁 | 有效 Lock-Token | 无体 | 204 |
| UNLOCK | 令牌不存在或路径不匹配 | 无效/不匹配令牌 | 错误响应 | 409/400 |
| UNLOCK | 权限不符 | 非持有者 | 错误响应 | 403/409 |

### 请求体解析与验证

实现通过结构体将 LOCK 请求体的关键元素映射为程序可处理的数据模型:

- LockRequest(建议在 handler.go 中补齐定义):用于严谨解析 lockscope(exclusive/shared)、locktype(write)、owner、timeout、depth。
- 校验逻辑:
  - lockscope 必须明确指定 exclusive 或 shared;locktype 应为 write。
  - timeout 支持 Second-xxx 与 infinite;缺省时采用服务端默认(例如 3600 秒)。
  - depth 支持 0 或 infinity;缺省建议为 0(仅资源自身)。
- 错误处理:
  - 请求体为空或解析失败返回 400,并记录日志。
  - 元素缺失或非法组合返回 400,并给出明确错误信息。

为便于客户端与服务端就解析策略达成一致,以下表格总结超时与深度的处理建议。

表 3:超时与深度解析策略
| 字段 | 合法值 | 默认值 | 处理逻辑 |
|---|---|---|---|
| timeout | Second-xxx、infinite | 3600 秒 | 解析为秒数;超期后自动清理 |
| depth | 0、infinity | 0 | 锁作用范围为资源自身或子树 |

### 锁定创建与令牌生成

- CreateLock:
  - 以路径、锁类型(EXCLUSIVE/SHARED)、owner(userID)、超时(秒)为参数创建锁。
  - 内部设置 CreatedAt、Depth、LockRoot 等元数据。
- 令牌生成:
  - 使用稳定前缀(例如 opaquelocktoken:)与路径、时间戳组合生成唯一令牌。
  - 令牌在响应头 Lock-Token 中回写,并在活动锁 discovery 中以 href 形式呈现。
- 超时机制:
  - 基于 CreatedAt 与 Timeout 计算过期时间;CleanExpiredLocks 在查询与响应前清理过期锁。
  - 刷新语义建议通过 If 头部的 timeout 语义在后续迭代中实现。

### 冲突检测与错误处理

冲突检测遵循 WebDAV 规范的基本原则,并结合当前实现的辅助方法:

- CheckExclusiveLock:
  - 若资源存在 EXCLUSIVE 锁且非持有者访问写入类操作,返回 423 Locked,并通过 SendLockedError 输出错误体。
- CheckSharedLock:
  - 允许在 SHARED 锁下进行读取;若尝试写入且存在冲突,则返回 423。
- CheckParentLocks:
  - 对目标路径的父目录进行锁检查;若父目录存在 EXCLUSIVE 锁且非持有者,则拒绝写入类操作。
- 统一检测:
  - 建议在 PUT、DELETE、MOVE、COPY、MKCOL 中统一调用 CheckAndEnforceLocks,确保行为一致。

表 4:冲突场景与处理
| 场景 | 检测方法 | 返回状态 | 响应体 |
|---|---|---|---|
| EXCLUSIVE 锁非持有者写入 | CheckExclusiveLock | 423 | LockedError |
| SHARED 锁写入冲突 | CheckSharedLock | 423 | LockedError |
| 父目录 EXCLUSIVE 锁 | CheckParentLocks | 423 | LockedError |
| UNLOCK 令牌不匹配 | GetLock/RemoveLock | 409/400 | 文本或 XML 错误 |

### 响应生成与 XML 输出

- LOCK 响应:
  - 主体结构包含 lockdiscovery(activelock)与 supportedlock(lockentry)。
  - activelock 包含 lockscope、locktype、depth、owner、timeout、locktoken(href)。
  - 头部:Content-Type: application/xml; charset=utf-8;Lock-Token 回写令牌。
- PROPFIND 集成:
  - supportedlock 与 lockdiscovery 已通过 ResponseProp 输出;需确保命名空间与元素前缀一致(例如 D: 前缀与 xmlns:D="DAV:")。
- XML 编码:
  - 使用 xml.Encoder 缩进输出;确保元素与命名空间严格对齐 RFC 4918。

表 5:响应 XML 元素与命名空间
| 元素 | 含义 | 命名空间 | 输出位置 |
|---|---|---|---|
| activelock | 活动锁 | DAV: | lockdiscovery |
| lockscope | 锁范围(exclusive/shared) | DAV: | activelock |
| locktype | 锁类型(write) | DAV: | activelock |
| locktoken.href | 锁令牌引用 | DAV: | activelock |
| owner | 锁持有者信息 | DAV: | activelock |
| timeout | 超时值 | DAV: | activelock |
| depth | 锁深度 | DAV: | activelock |
| supportedlock | 支持的锁能力 | DAV: | 响应属性 |
| lockentry | 锁能力条目 | DAV: | supportedlock |

## 与现有 HTTP 方法的集成与回归

锁定机制需要与现有方法协同工作,确保读取与写入行为在锁约束下保持一致与可预期。

- GET/HEAD:
  - 在 SHARED 锁下允许读取;在 EXCLUSIVE 锁下仅锁持有者可读取。
  - 当前实现已调用 CheckSharedLock;需确认在 EXCLUSIVE 场景下对持有者的判断逻辑是否完善。
- PUT/DELETE/MOVE/COPY/MKCOL:
  - 写入类操作需满足锁约束;建议统一调用 CheckAndEnforceLocks,在存在冲突时返回 423。
  - MOVE/COPY 的目标与源路径均需满足锁约束,避免产生不一致状态。
- PROPFIND:
  - supportedlock 与 lockdiscovery 已输出;需验证在锁过期清理后响应准确。
- OPTIONS:
  - Allow 头部已包含 LOCK/UNLOCK;路由接入后需确保方法可达。

表 6:方法交互矩阵
| 方法 | 锁定约束 | 预期行为 | 当前状态 |
|---|---|---|---|
| GET | SHARED 可读;EXCLUSIVE 仅持有者可读 | 200/资源内容 | 已调用 CheckSharedLock |
| HEAD | 同 GET | 200/头部 | 已调用 CheckSharedLock |
| PUT | 需持有 EXCLUSIVE 或无冲突 SHARED | 201/成功;423/冲突 | 待统一调用 CheckAndEnforceLocks |
| DELETE | 需满足锁约束 | 204/成功;423/冲突 | 待统一调用 CheckAndEnforceLocks |
| MOVE/COPY | 目标与源均需满足锁约束 | 201/成功;409/路径冲突;423/锁冲突 | 待统一调用 CheckAndEnforceLocks |
| MKCOL | 需满足父目录锁约束 | 201/成功;409/路径冲突;423/锁冲突 | 待统一调用 CheckAndEnforceLocks |
| PROPFIND | 无写入约束 | 200 + supportedlock/lockdiscovery | 已实现 |
| LOCK/UNLOCK | 令牌与权限校验 | 200/204/423/400/409 | 已实现(需接入与验证) |

## 测试计划与验证

为确保实现的正确性与规范性,测试覆盖功能、规范对齐、性能与回归四个维度:

- 功能测试:
  - LOCK 请求体解析与错误场景(非法 XML、缺少元素、非法组合)。
  - 锁创建、令牌回写、UNLOCK 权限校验与幂等性。
  - 冲突检测(EXCLUSIVE/SHARED、父目录锁)。
- 规范对齐:
  - 423/409/400/204 返回码与响应体格式。
  - XML 元素与命名空间一致性。
- 性能与稳定性:
  - 并发锁创建与解锁场景下的正确性与安全性。
  - 过期清理机制在高并发下的表现。
- 回归测试:
  - PROPFIND supportedlock/lockdiscovery 输出不受影响。
  - GET/HEAD/PUT/DELETE/MOVE/COPY/MKCOL 行为与现有预期一致。

表 7:测试用例总览
| 用例ID | 场景 | 请求 | 期望响应 | 验证点 |
|---|---|---|---|---|
| T1 | LOCK 解析失败 | 非法 XML | 400 | 错误日志与提示 |
| T2 | 创建 EXCLUSIVE 锁 | 合法请求 | 200 + Lock-Token | 令牌唯一性、activelock 完整 |
| T3 | 创建 SHARED 锁 | 合法请求 | 200 + Lock-Token | SHARED 与 EXCLUSIVE 互斥 |
| T4 | EXCLUSIVE 锁冲突 | 非持有者写入 | 423 | LockedError 元素与命名空间 |
| T5 | 父目录锁冲突 | 子资源写入 | 423 | CheckParentLocks 生效 |
| T6 | UNLOCK 幂等 | 重复解锁 | 204 | 锁移除成功 |
| T7 | UNLOCK 权限不符 | 非持有者解锁 | 423/409 | 权限校验正确 |
| T8 | PROPFIND lockdiscovery | 有活动锁 | 200 + XML | ActiveLock 元素正确 |
| T9 | 并发创建锁 | 多用户并发 | 200/423 | 锁管理器并发安全 |
| T10 | 过期清理 | 超时后访问 | 200/423 | CleanExpiredLocks 生效 |

## 风险与信息缺口

- 路由接入风险:handleLock/handleUnlock 已实现,但需在路由层显式注册并验证可达性。
- 规范对齐风险:LockedError 的元素与命名空间需严格对照 RFC 4918,避免客户端互操作问题。
- 并发与性能风险:锁管理器在极端并发下可能存在竞态;过期清理频率与锁数量增长对内存的影响需监控。
- 存储后端差异:StatObject/ListObjects 行为差异可能影响锁冲突判断与 PROPFIND 响应。
- 日志与可观测性:关键路径需补充结构化日志与错误包装,便于调试与运维。

信息缺口与待决议事项:
- 路由注册:需确认 handleLock/handleUnlock 在路由层的绑定语句与路径规则。
- LockRequest 定义:建议在 handler.go 中补齐,确保解析严谨性与可测试性。
- 超时与刷新细化:Timeout 头部解析与 If.timeout 刷新语义需进一步实现与文档化。
- 错误响应对齐:LockedError 的元素与命名空间需对齐 RFC 4918。
- 冲突检测统一:写入类方法建议统一调用 CheckAndEnforceLocks,需评估性能与兼容性。
- 日志与错误处理:关键路径需补充结构化日志与错误包装。

## 后续迭代与优化建议

- 路由接入与联调:完成 handleLock/handleUnlock 的路由注册,开展跨客户端联调与回归。
- 超时与刷新:细化 Timeout 解析策略与 If.timeout 语义;在带 If 的请求中自动延长锁有效期。
- 冲突检测统一:在 PUT/DELETE/MOVE/COPY/MKCOL 中统一调用 CheckAndEnforceLocks,形成一致的约束模型。
- 错误响应规范:对齐 RFC 4918 的错误体元素与命名空间;在 423 中回写 Lock-Token 与 Retry-After。
- 日志与监控:补充结构化日志(路径、用户、锁类型、令牌、超时、冲突类型),增加指标(锁定数量、过期清理次数、刷新次数)。

## 结论

当前实现已具备 LOCK/UNLOCK 的核心能力:请求解析、锁创建与验证、令牌生成、超时与过期清理、冲突检测与错误处理、标准化响应 XML 与 PROPFIND 集成。方法层面已覆盖 GET/HEAD/PUT/DELETE/MKCOL/MOVE/COPY/PROPFIND/OPTIONS,OPTIONS 头部已包含 LOCK/UNLOCK。下一步需完成路由接入与联调,细化超时与刷新语义,统一写入类方法的冲突检测,并在错误响应与日志可观测性上对齐 RFC 4918 与工程实践。通过系统的测试与回归,本实现能够满足工程团队对稳定性、规范性与可维护性的要求。

## 附录:关键结构体与方法清单

- 结构体:
  - LockManager:锁创建、查询、移除、冲突检测、过期清理。
  - Lock、ActiveLock、LockInfo、LockScopeInfo、LockTypeInfo、LockToken:锁数据与响应元素。
  - LockedError:423 错误响应体。
  - ResponseProp:PROPFIND 响应属性,包含 supportedlock 与 lockdiscovery。
- 关键方法:
  - LockManager:CreateLock、GetLock、GetLocksForPath、RemoveLock、CheckLock、CheckExclusiveLock、CheckParentLocks、GetLockDiscovery、CleanExpiredLocks。
  - Handler:HandleLock、HandleUnlock、SendLockedError、CheckExclusiveLock、CheckSharedLock、CheckParentLocks、CheckAnyLock、CheckAndEnforceLocks、createFileResponse、createFolderResponse。
- 命名空间与元素:
  - DAV: 作为主命名空间;lockscope、locktype、activelock、locktoken(href)、owner、timeout、depth、supportedlock(lockentry)。

表 8:接口清单与职责
| 接口/结构体 | 职责 | 输入 | 输出 | 所在文件 |
|---|---|---|---|---|
| LockManager | 锁生命周期与冲突检测 | 路径/类型/owner/超时 | 锁对象/冲突结果 | lock.go |
| Lock | 锁数据 | - | 锁字段 | lock.go |
| ActiveLock | 活动锁 XML | - | XML 元素 | lock.go |
| LockInfo/LockScopeInfo/LockTypeInfo/LockToken | 响应元素 | - | XML 元素 | lock.go |
| LockedError | 423 错误体 | 锁信息 | XML 错误 | handler.go |
| Handler.HandleLock | 处理 LOCK 请求 | 请求体/头部 | 200 + XML | handler.go |
| Handler.HandleUnlock | 处理 UNLOCK 请求 | Lock-Token | 204/错误 | handler.go |
| Handler.SendLockedError | 发送 423 | 锁信息 | XML 错误 | handler.go |
| Handler.CheckAndEnforceLocks | 统一冲突检测 | 路径 | 布尔/错误 | handler.go |
| Handler.createFileResponse/createFolderResponse | PROPFIND 响应 | 资源元数据 | XML 响应 | handler.go |

---

信息缺口与待决议事项(汇总):
- 路由接入:需在路由层确认 handleLock/handleUnlock 的注册语句与路径规则。
- LockRequest:建议在 handler.go 中补齐定义,提升解析严谨性与可测试性。
- 超时与刷新:细化 Timeout 头部解析与 If.timeout 刷新语义,并补充实现。
- 错误响应对齐:LockedError 的元素与命名空间需严格对齐 RFC 4918。
- 冲突检测统一:在写入类方法中统一调用 CheckAndEnforceLocks,评估性能与兼容性。
- 日志与错误处理:补充结构化日志与错误包装,提升可观测性与调试效率。

本报告为工程团队提供了清晰的实现蓝图与验证路径,后续应围绕上述信息缺口制定具体任务并逐步落地。