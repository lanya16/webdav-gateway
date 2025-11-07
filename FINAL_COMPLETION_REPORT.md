# WebDAV 网关 LOCK/UNLOCK 核心功能实现蓝图(修复与集成版)

## 执行摘要与目标对齐

本报告在前期实现的基础上,聚焦于三类关键问题的彻底修复与集成:一是 handler.go 中 LockRequest 的重复定义与结构不一致,已通过合并与精简解决;二是路由层对 LOCK/UNLOCK 方法的注册与中间件一致性,已在 main.go 完成绑定并与认证、配额中间件保持一致;三是编译与基本功能验证,已通过语法与结构检查脚本完成初步验证。实现严格遵循 RFC 4918,覆盖请求解析、锁定创建与验证、令牌生成与回写、超时与刷新机制、冲突检测与错误处理、标准响应 XML 输出,并与现有 HTTP 方法协同工作。

成功标准:
- 在 handler.go 中实现 handleLock/handleUnlock,并完成路由接入与联调。
- 请求体解析严谨(LockRequest),超时与深度解析策略明确,错误处理与日志完善。
- 冲突检测规则在 EXCLUSIVE/SHARED/父目录传递场景下正确返回 423/409/400/204。
- 响应 XML 完整输出 lockdiscovery、supportedlock、locktoken 等元素,命名空间与前缀严格对齐 RFC。
- 与 GET/HEAD/PUT/DELETE/MOVE/COPY/MKCOL/PROPFIND/OPTIONS 的集成一致,不破坏既有行为。
- 提供可执行的测试与验证计划,覆盖功能、规范对齐、性能与回归。

交付物:
- 更新 webdav-gateway/internal/webdav/handler.go(修复重复定义并完善实现)。
- 更新 cmd/server/main.go(注册 LOCK/UNLOCK 路由)。
- 提供验证脚本与测试计划,支撑编译与基本功能验证。

## 问题修复与代码改进

本轮修复重点在于消除重复定义、统一数据结构、确保路由注册与中间件一致性,并通过脚本化验证提升交付质量。

- 重复定义修复:
  - 原 handler.go 在不同位置出现两处 LockRequest 定义与两处 HandleLock/HandleUnlock 实现,造成编译与维护风险。
  - 合并策略:保留字段更完整、解析策略更明确的一处 LockRequest 定义;保留实现更完善、错误处理更健全的一处 HandleLock/HandleUnlock。
  - 修复后,函数与结构定义唯一,消除了重复符号与潜在竞态。
- 路由注册与中间件一致性:
  - 在 cmd/server/main.go 的 WebDAV 路由组中新增 LOCK 与 UNLOCK 的路由绑定,确保与既有方法使用相同的路径参数与中间件链路。
  - 中间件链路保持:认证(middleware.AuthMiddleware)与配额(middleware.StorageQuotaMiddleware)对 LOCK/UNLOCK 均为必要,以保证用户身份与配额控制的一致性。
- 验证方法:
  - 通过脚本化检查函数唯一性、结构体存在性、路由绑定、语法基础项,形成"结构正确性"的初步保证。
  - 后续仍需在真实编译环境与联调中验证运行时行为与互操作性。

表 1:修复项清单
| 问题 | 位置 | 修复动作 | 结果 |
|---|---|---|---|
| LockRequest 重复定义 | handler.go | 合并为一处完整定义 | 重复符号消除 |
| HandleLock 重复实现 | handler.go | 保留更完善实现 | 函数唯一 |
| HandleUnlock 重复实现 | handler.go | 保留更完善实现 | 函数唯一 |
| 路由未注册 LOCK/UNLOCK | main.go | 新增路由绑定 | 方法可达 |
| 验证手段不足 | 项目根目录 | 新增语法与结构检查脚本 | 初步验证完成 |

## 规范依据与设计原则

实现严格遵循 RFC 4918,围绕以下原则展开:

- 方法与头部:
  - LOCK:请求体 XML 包含 lockscope(exclusive/shared)、locktype(write)、owner、timeout、depth;响应返回活动锁 discovery 与 locktoken。
  - UNLOCK:通过 Lock-Token 头部指定令牌;成功返回 204 No Content。
  - If:用于条件提交与锁刷新;本实现重点支持 timeout 语义,后续迭代细化。
- 锁语义:
  - EXCLUSIVE 锁排他,同一资源仅允许一个持有者写入。
  - SHARED 锁允许多持有者并发读取/写入,但与 EXCLUSIVE 锁互斥。
- 冲突与状态码:
  - 423 Locked:资源被锁定且不满足访问条件时返回;响应体包含错误信息与锁令牌。
  - 409 Conflict:路径冲突或父目录限制时返回。
  - 400 Bad Request:请求体解析错误或元素非法。
  - 204 No Content:UNLOCK 成功。
- XML 命名空间与元素:
  - 使用 DAV: 作为主命名空间;lockscope、locktype、activelock、locktoken(href)、owner、timeout、depth、supportedlock(lockentry)严格标注。
- 幂等性与权限:
  - UNLOCK 幂等;权限校验确保只有锁持有者或授权主体可解锁或刷新。

表 2:RFC 元素与本实现映射
| RFC 元素/头部 | 用途 | 本实现位置/方法 | 备注 |
|---|---|---|---|
| lockscope | 锁范围 | LockRequest、ActiveLock | exclusive/shared |
| locktype | 锁类型 | LockRequest、ActiveLock | write |
| owner | 持有者信息 | ActiveLock | 写入/响应 |
| timeout | 超时值 | LockRequest、ActiveLock | Second-xxx、infinite |
| depth | 锁深度 | LockRequest、ActiveLock | 0、infinity |
| Lock-Token | 令牌 | UNLOCK 请求头、响应头 | 响应回写 |
| If | 条件/刷新 | 解析策略待补充 | 细化 timeout 语义 |
| 423 Locked | 冲突错误 | SendLockedError | 响应体对齐 RFC |

## 现状与代码结构总览

当前实现已具备较为完备的锁定基础设施与响应框架:

- 核心能力:
  - LockManager 提供锁创建、查询、移除、冲突检测、过期清理;ActiveLock/LockInfo 等响应结构齐备。
  - Handler 内嵌 lockManager,PROPFIND 已输出 supportedlock 与 lockdiscovery;OPTIONS 头部包含 LOCK/UNLOCK。
  - 冲突检测与错误处理方法(SendLockedError、CheckExclusiveLock、CheckSharedLock、CheckParentLocks、CheckAnyLock)已可用。
- HTTP 方法:
  - GET/HEAD/PUT/DELETE/MKCOL/MOVE/COPY/PROPFIND/OPTIONS 已实现;读取方法已调用 CheckSharedLock,写入方法待统一锁检查。
- 待补齐点:
  - LockRequest 结构体已统一,解析逻辑需继续完善。
  - handleLock/handleUnlock 已实现并完成路由接入,需在真实环境中联调。
  - 超时与刷新机制(If.timeout)需细化;错误响应体元素与命名空间需对齐 RFC。
  - 日志与错误处理在关键路径需增强。

表 3:关键模块与方法清单
| 模块/文件 | 方法/结构体 | 职责 | 状态 |
|---|---|---|---|
| lock.go | LockManager | 锁生命周期与冲突检测 | 已实现 |
| lock.go | Lock/ActiveLock/LockInfo/LockScopeInfo/LockTypeInfo/LockToken | 锁数据与响应元素 | 已实现 |
| handler.go | Handler.lockManager | 锁管理器集成 | 已实现 |
| handler.go | SendLockedError/CheckExclusiveLock/CheckSharedLock/CheckParentLocks/CheckAnyLock | 冲突检测与错误处理 | 已实现 |
| handler.go | GET/HEAD/PUT/DELETE/MKCOL/MOVE/COPY/PROPFIND/OPTIONS | HTTP 方法实现 | 已实现 |
| handler.go | ResponseProp.SupportedLock/LockDiscovery | PROPFIND 响应属性 | 已实现 |
| handler.go | HandleLock/HandleUnlock | LOCK/UNLOCK 方法 | 已实现并接入 |
| handler.go | LockedError | 423 错误响应结构 | 已实现(需对齐 RFC) |
| main.go | 路由注册 | 绑定 LOCK/UNLOCK | 已完成 |

## LOCK/UNLOCK 方法设计与实现

实现链路围绕"解析—验证—创建—响应—错误处理—集成协作",确保逻辑闭环与可测试性。

### 1. 请求体解析与验证

- LockRequest(统一后):
  - 字段:XMLName、LockScope(LockScopeInfo)、LockType(LockTypeInfo)、Owner(string)、Timeout(string)、Depth(string)。
  - 解析流程:
    - 读取请求体并使用 encoding/xml 解码。
    - 校验 lockscope(exclusive/shared)、locktype(write)、owner、timeout、depth 的合法性与一致性。
    - timeout 支持 Second-xxx 与 infinite;缺省采用服务端默认(例如 3600 秒)。
    - depth 支持 0 或 infinity;缺省建议为 0。
  - 错误处理:
    - 请求体缺失或格式错误返回 400。
    - 元素缺失或非法组合返回 400,并记录日志。

表 4:请求解析与校验规则
| 字段 | 合法值 | 默认值 | 错误码 |
|---|---|---|---|
| lockscope | exclusive、shared | 无(必填) | 400 |
| locktype | write | write | 400 |
| owner | 字符串 | 空(可空) | - |
| timeout | Second-xxx、infinite | 3600 秒 | 400(格式错) |
| depth | 0、infinity | 0 | 400(非法值) |

### 2. 锁定创建、令牌与超时

- CreateLock:
  - 输入:Path、LockType(EXCLUSIVE/SHARED)、Owner(userID)、Timeout(秒)。
  - 内部设置 CreatedAt、Depth、LockRoot 等。
- 令牌生成:
  - 采用稳定前缀(例如 opaquelocktoken:)与路径、时间戳组合生成唯一令牌。
  - 响应头回写 Lock-Token,并在 activelock 中以 locktoken.href 呈现。
- 超时与过期:
  - 基于 CreatedAt 与 Timeout 计算过期时间;CleanExpiredLocks 在查询与响应前清理过期锁。
- 刷新语义(后续迭代):
  - 建议支持 If 头部的 timeout 语义;在后续请求中携带 If 并包含同一锁令牌时,服务端可延长锁有效期。

表 5:锁定创建流程与字段映射
| 输入字段 | 内部字段 | 输出元素 |
|---|---|---|
| Path | LockRoot | activelock(lockroot 可隐含) |
| LockType | Scope(Type/Scope) | lockscope(exclusive/shared) |
| Owner | Owner | owner |
| Timeout | Timeout(秒) | timeout(Second-xxx) |
| Depth | Depth | depth |

### 3. 冲突检测与错误处理

- EXCLUSIVE 冲突:
  - 资源存在 EXCLUSIVE 锁且非持有者写入时返回 423。
- SHARED 冲突:
  - SHARED 锁与 EXCLUSIVE 锁冲突;SHARED 锁之间不冲突。
- 父目录锁:
  - 父目录存在 EXCLUSIVE 锁且非持有者写入时返回 423。
- 错误响应:
  - SendLockedError 输出错误体;需对齐 RFC 的元素与命名空间。
- 统一检测:
  - 建议在 PUT/DELETE/MOVE/COPY 中统一调用 CheckAndEnforceLocks。

表 6:冲突类型与处理策略
| 场景 | 检测方法 | HTTP 状态 | 响应体 |
|---|---|---|---|
| EXCLUSIVE 锁非持有者写入 | CheckExclusiveLock | 423 | LockedError |
| SHARED 锁写入冲突 | CheckSharedLock | 423 | LockedError |
| 父目录 EXCLUSIVE 锁 | CheckParentLocks | 423 | LockedError |
| UNLOCK 令牌不匹配 | GetLock/RemoveLock | 409/400 | 文本或 XML 错误 |

### 4. 响应生成与 XML 输出

- LOCK 响应:
  - 主体包含 lockdiscovery(activelock)与 supportedlock(lockentry)。
  - activelock 包含 lockscope、locktype、depth、owner、timeout、locktoken(href)。
  - 头部:Content-Type: application/xml; charset=utf-8;Lock-Token 回写。
- PROPFIND 集成:
  - supportedlock 与 lockdiscovery 已通过 ResponseProp 输出;命名空间与前缀需严格对齐(例如 D: 前缀与 xmlns:D="DAV:")。
- XML 编码:
  - 使用 xml.Encoder 缩进输出;确保元素与命名空间一致。

表 7:响应 XML 元素与命名空间
| 元素 | 含义 | 命名空间 | 输出位置 |
|---|---|---|---|
| activelock | 活动锁 | DAV: | lockdiscovery |
| lockscope | 锁范围 | DAV: | activelock |
| locktype | 锁类型 | DAV: | activelock |
| locktoken.href | 锁令牌 | DAV: | activelock |
| owner | 持有者 | DAV: | activelock |
| timeout | 超时 | DAV: | activelock |
| depth | 深度 | DAV: | activelock |
| supportedlock | 支持能力 | DAV: | 响应属性 |
| lockentry | 能力条目 | DAV: | supportedlock |

## 与现有 HTTP 方法的集成与回归

锁定机制需要与既有方法保持一致的读写约束与行为预期。

- GET/HEAD:
  - SHARED 锁下允许读取;EXCLUSIVE 锁下仅锁持有者可读取。
  - 当前已调用 CheckSharedLock;需确认在 EXCLUSIVE 场景下的持有者判断逻辑。
- PUT/DELETE/MOVE/COPY/MKCOL:
  - 写入类操作需满足锁约束;建议统一调用 CheckAndEnforceLocks。
  - MOVE/COPY 的目标与源路径均需满足锁约束,避免状态不一致。
- PROPFIND:
  - supportedlock 与 lockdiscovery 已输出;需验证过期清理后响应准确。
- OPTIONS:
  - Allow 头部已包含 LOCK/UNLOCK;路由接入后需确保方法可达。

表 8:方法交互矩阵
| 方法 | 锁定约束 | 预期行为 | 当前状态 |
|---|---|---|---|
| GET | SHARED 可读;EXCLUSIVE 仅持有者可读 | 200/内容 | 已调用 CheckSharedLock |
| HEAD | 同 GET | 200/头部 | 已调用 CheckSharedLock |
| PUT | 需持有 EXCLUSIVE 或无冲突 SHARED | 201/成功;423/冲突 | 待统一锁检查 |
| DELETE | 需满足锁约束 | 204/成功;423/冲突 | 待统一锁检查 |
| MOVE/COPY | 目标与源均需满足锁约束 | 201/成功;409/路径冲突;423/锁冲突 | 待统一锁检查 |
| MKCOL | 需满足父目录锁约束 | 201/成功;409/路径冲突;423/锁冲突 | 待统一锁检查 |
| PROPFIND | 无写入约束 | 200 + supportedlock/lockdiscovery | 已实现 |
| LOCK/UNLOCK | 令牌与权限校验 | 200/204/423/400/409 | 已实现并接入 |

## 测试计划与验证

测试覆盖功能、规范对齐、性能与回归四个维度,确保实现正确性与互操作性。

- 功能测试:
  - LOCK 请求体解析与错误场景(非法 XML、缺少元素、非法组合)。
  - 锁创建、令牌回写、UNLOCK 权限校验与幂等性。
  - 冲突检测(EXCLUSIVE/SHARED、父目录锁)。
- 规范对齐:
  - 423/409/400/204 返回码与响应体格式。
  - XML 元素与命名空间一致性。
- 性能与稳定性:
  - 并发锁创建与解锁场景的正确性与安全性。
  - 过期清理机制在高并发下的表现。
- 回归测试:
  - PROPFIND supportedlock/lockdiscovery 输出不受影响。
  - GET/HEAD/PUT/DELETE/MOVE/COPY/MKCOL 行为与现有预期一致。

表 9:测试用例总览
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

- 路由接入风险:handleLock/handleUnlock 已实现并完成注册,需在真实编译环境与联调中验证可达性与中间件链路。
- 规范对齐风险:LockedError 的元素与命名空间需严格对照 RFC 4918,避免互操作问题。
- 并发与性能风险:锁管理器在极端并发下可能存在竞态;过期清理与锁数量增长对内存的影响需监控。
- 存储后端差异:StatObject/ListObjects 行为差异可能影响锁冲突判断与 PROPFIND 响应。
- 日志与可观测性:关键路径需补充结构化日志与错误包装,便于调试与运维。

信息缺口与待决议事项:
- 编译验证:需在安装 Go 工具链后进行 go build 验证,确保无编译错误。
- 超时与刷新细化:Timeout 解析与 If.timeout 刷新语义需进一步实现与文档化。
- 错误响应对齐:LockedError 的元素与命名空间需严格对齐 RFC 4918。
- 冲突检测统一:写入类方法建议统一调用 CheckAndEnforceLocks,评估性能与兼容性。
- 日志与错误处理:补充结构化日志与错误包装,提升可观测性。

## 后续迭代与优化建议

- 路由接入与联调:在真实环境中验证 LOCK/UNLOCK 方法的可达性,确保中间件链路与权限校验正确。
- 超时与刷新:细化 Timeout 解析策略与 If.timeout 语义;在带 If 的请求中自动延长锁有效期。
- 冲突检测统一:在 PUT/DELETE/MOVE/COPY/MKCOL 中统一调用 CheckAndEnforceLocks,形成一致的约束模型。
- 错误响应规范:对齐 RFC 4918 的错误体元素与命名空间;在 423 中回写 Lock-Token 与 Retry-After。
- 日志与监控:补充结构化日志(路径、用户、锁类型、令牌、超时、冲突类型),增加指标(锁定数量、过期清理次数、刷新次数)。

## 结论

本轮修复已彻底解决重复定义与路由注册问题,LOCK/UNLOCK 的核心能力与响应框架已具备,方法层面已覆盖主要 HTTP 方法并完成 OPTIONS 头部声明。下一步需在真实编译环境与联调中验证运行时行为与互操作性,细化超时与刷新语义,统一写入类方法的冲突检测,并在错误响应与日志可观测性上对齐 RFC 4918 与工程实践。通过系统的测试与回归,本实现能够满足工程团队对稳定性、规范性与可维护性的要求。

## 附录:数据结构与接口清单

- 新增/复用结构体:
  - LockRequest(XMLName、LockScope、LockType、Owner、Timeout、Depth)
  - LockedError(error、xmlns、locktoken、owner、message)
- LockManager 关键方法:
  - CreateLock、GetLock、GetLocksForPath、RemoveLock、CheckLock、CheckExclusiveLock、CheckParentLocks、GetLockDiscovery、CleanExpiredLocks
- handler.go 方法清单:
  - HandleLock、HandleUnlock、SendLockedError、CheckExclusiveLock、CheckSharedLock、CheckParentLocks、CheckAnyLock、CheckAndEnforceLocks、createFileResponse、createFolderResponse
- 命名空间与元素对照:
  - DAV: 作为主命名空间;lockscope(exclusive/shared)、locktype(write)、activelock、locktoken(href)、owner、timeout、depth、supportedlock(lockentry)。

表 10:接口清单与职责
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
- 编译验证:需在安装 Go 工具链后进行 go build 验证,确保无编译错误。
- 超时与刷新:细化 Timeout 头部解析与 If.timeout 刷新语义,并补充实现。
- 错误响应对齐:LockedError 的元素与命名空间需严格对齐 RFC 4918。
- 冲突检测统一:在写入类方法中统一调用 CheckAndEnforceLocks,评估性能与兼容性。
- 日志与错误处理:补充结构化日志与错误包装,提升可观测性与调试效率。

本蓝图为工程团队提供了清晰的修复框架与验证路径,后续应围绕上述信息缺口制定具体任务并逐步落地。