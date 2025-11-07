package webdav

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/webdav-gateway/internal/config"
)

// LockType 定义锁定类型
type LockType string

const (
	LockTypeExclusive LockType = "exclusive"
	LockTypeShared    LockType = "shared"
)

// LockScope 定义锁定范围
type LockScope string

const (
	LockScopeExclusive LockScope = "exclusive"
	LockScopeShared    LockScope = "shared"
)

// Lock 锁定信息结构
type Lock struct {
	Token       string    `json:"token"`
	Type        LockType  `json:"type"`
	Scope       LockScope `json:"scope"`
	Owner       string    `json:"owner"`
	Timeout     int64     `json:"timeout"` // 秒
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Path        string    `json:"path"`
	Depth       int       `json:"depth"`     // 0 或 infinity (用-1表示)
	LockRoot    string    `json:"lock_root"` // 锁根路径
	RefreshHint time.Duration `json:"refresh_hint,omitempty"`
}

// LockEntry 锁定信息用于响应
type LockEntry struct {
	XMLName   xml.Name      `xml:"D:lockentry"`
	LockScope LockScopeInfo `xml:"D:lockscope"`
	LockType  LockTypeInfo  `xml:"D:locktype"`
}

type LockScopeInfo struct {
	Exclusive *struct{} `xml:"D:exclusive,omitempty"`
	Shared    *struct{} `xml:"D:shared,omitempty"`
}

type LockTypeInfo struct {
	Write *struct{} `xml:"D:write,omitempty"`
}

type ActiveLock struct {
	XMLName   xml.Name      `xml:"D:activelock"`
	LockScope LockScopeInfo `xml:"D:lockscope"`
	LockType  LockTypeInfo  `xml:"D:locktype"`
	Depth     string        `xml:"D:depth"`
	Owner     string        `xml:"D:owner,omitempty"`
	Timeout   string        `xml:"D:timeout"`
	LockToken LockToken     `xml:"D:locktoken"`
	LockRoot  LockRoot      `xml:"D:lockroot"`
}

type LockToken struct {
	Href string `xml:"D:href"`
}

type LockRoot struct {
	Href string `xml:"D:href"`
}

// LockManager 锁定管理器
type LockManager struct {
	locks       map[string]*Lock   // token -> Lock
	locksByPath map[string][]*Lock // path -> []*Lock
	mu          sync.RWMutex
	maxTimeout  int64 // 最大超时时间（秒）
	
	// 持久化和备份功能
	persistence *LockPersistence
	backup      *LockBackup
	config      *config.LockPersistenceConfig
	lastSync    time.Time
}

// NewLockManager 创建新的锁定管理器
func NewLockManager() *LockManager {
	return NewLockManagerWithConfig(nil)
}

// NewLockManagerWithConfig 创建新的锁定管理器（带配置）
func NewLockManagerWithConfig(lockConfig *config.LockPersistenceConfig) *LockManager {
	lm := &LockManager{
		locks:       make(map[string]*Lock),
		locksByPath: make(map[string][]*Lock),
		maxTimeout:  86400, // 默认最大超时24小时
		config:      lockConfig,
	}

	// 如果配置了持久化，初始化持久化管理器
	if lockConfig != nil && lockConfig.Enabled {
		persistence, err := NewLockPersistence(lockConfig)
		if err != nil {
			log.Printf("Warning: failed to initialize lock persistence: %v", err)
		} else {
			lm.persistence = persistence
			
			// 初始化备份管理器
			lm.backup = NewLockBackup(persistence, lockConfig)
			
			// 从持久化存储恢复锁定数据
			if err := lm.restoreFromPersistence(); err != nil {
				log.Printf("Warning: failed to restore locks from persistence: %v", err)
			}
			
			// 启动自动备份
			lm.backup.StartAutoBackup()
		}
	}

	// 启动后台清理任务
	go lm.startCleanupTask()
	
	// 如果启用了持久化，启动同步任务
	if lm.persistence != nil {
		go lm.startSyncTask()
	}

	return lm
}

// generateLockToken 生成唯一的锁定令牌
func (lm *LockManager) generateLockToken() string {
	// 使用 UUID v4 生成唯一令牌
	u := uuid.New()
	return fmt.Sprintf("opaquelocktoken:%s", u.String())
}

// generateSecureToken 生成加密安全的随机令牌（备用方案）
func (lm *LockManager) generateSecureToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 如果加密随机失败，回退到UUID
		return lm.generateLockToken()
	}
	return fmt.Sprintf("opaquelocktoken:%s", hex.EncodeToString(b))
}

// CreateLock 创建锁定
func (lm *LockManager) CreateLock(path string, lockType LockType, owner string, timeout int64, depth int) *Lock {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// 应用超时上限
	if timeout > lm.maxTimeout || timeout <= 0 {
		timeout = lm.maxTimeout
	}

	token := lm.generateLockToken()
	now := time.Now()

	lock := &Lock{
		Token:       token,
		Type:        lockType,
		Scope:       LockScope(lockType),
		Owner:       owner,
		Timeout:     timeout,
		CreatedAt:   now,
		ExpiresAt:   now.Add(time.Duration(timeout) * time.Second),
		Path:        path,
		Depth:       depth,
		LockRoot:    path,
		RefreshHint: time.Duration(timeout/2) * time.Second, // 建议在一半时间后刷新
	}

	lm.locks[token] = lock
	lm.locksByPath[path] = append(lm.locksByPath[path], lock)

	// 持久化锁定
	if lm.persistence != nil {
		if err := lm.persistence.SaveLock(lock); err != nil {
			log.Printf("Warning: failed to persist lock: %v", err)
		}
	}

	return lock
}

// RefreshLock 刷新锁定
func (lm *LockManager) RefreshLock(token string, timeout int64) (*Lock, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lock, exists := lm.locks[token]
	if !exists {
		return nil, fmt.Errorf("lock token not found")
	}

	// 检查锁定是否已过期
	if time.Now().After(lock.ExpiresAt) {
		lm.removeLockUnsafe(token)
		return nil, fmt.Errorf("lock has expired")
	}

	// 应用超时上限
	if timeout > lm.maxTimeout || timeout <= 0 {
		timeout = lm.maxTimeout
	}

	// 更新超时
	now := time.Now()
	lock.Timeout = timeout
	lock.ExpiresAt = now.Add(time.Duration(timeout) * time.Second)
	lock.RefreshHint = time.Duration(timeout/2) * time.Second

	// 持久化更新
	if lm.persistence != nil {
		if err := lm.persistence.SaveLock(lock); err != nil {
			log.Printf("Warning: failed to persist refreshed lock: %v", err)
		}
	}

	return lock, nil
}

// GetLock 获取锁定信息
func (lm *LockManager) GetLock(token string) (*Lock, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	lock, exists := lm.locks[token]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(lock.ExpiresAt) {
		return nil, false
	}

	return lock, true
}

// GetLocksForPath 获取路径的所有锁定
func (lm *LockManager) GetLocksForPath(path string) []*Lock {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	var validLocks []*Lock
	locks := lm.locksByPath[path]

	for _, lock := range locks {
		// 只返回未过期的锁
		if time.Now().Before(lock.ExpiresAt) {
			validLocks = append(validLocks, lock)
		}
	}

	return validLocks
}

// GetLockForPathAndUser 获取路径上特定用户的锁定
func (lm *LockManager) GetLockForPathAndUser(path, userID string) *Lock {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	locks := lm.locksByPath[path]

	for _, lock := range locks {
		// 只返回未过期的锁
		if time.Now().Before(lock.ExpiresAt) && lock.Owner == userID {
			return lock
		}
	}

	return nil
}

// RemoveLock 移除锁定
func (lm *LockManager) RemoveLock(token string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	result := lm.removeLockUnsafe(token)
	
	// 从持久化存储中删除
	if result && lm.persistence != nil {
		if err := lm.persistence.DeleteLock(token); err != nil {
			log.Printf("Warning: failed to delete lock from persistence: %v", err)
		}
	}
	
	return result
}

// removeLockUnsafe 不加锁的移除锁定（内部使用）
func (lm *LockManager) removeLockUnsafe(token string) bool {
	lock, exists := lm.locks[token]
	if !exists {
		return false
	}

	// 从路径映射中移除
	if locks, ok := lm.locksByPath[lock.Path]; ok {
		for i, l := range locks {
			if l.Token == token {
				lm.locksByPath[lock.Path] = append(locks[:i], locks[i+1:]...)
				break
			}
		}
		// 如果路径下没有锁了，删除该路径的映射
		if len(lm.locksByPath[lock.Path]) == 0 {
			delete(lm.locksByPath, lock.Path)
		}
	}

	delete(lm.locks, token)
	return true
}

// CheckLock 检查路径的锁定状态
func (lm *LockManager) CheckLock(path string, userID string) (bool, *Lock, error) {
	locks := lm.GetLocksForPath(path)

	for _, lock := range locks {
		// 如果是排他锁且不是持有者
		if lock.Type == LockTypeExclusive && lock.Owner != userID {
			return true, lock, fmt.Errorf("resource is locked exclusively by %s", lock.Owner)
		}
	}

	return false, nil, nil
}

// CheckExclusiveLock 检查是否被排他锁定
func (lm *LockManager) CheckExclusiveLock(path string, userID string) (bool, *Lock, error) {
	locks := lm.GetLocksForPath(path)

	for _, lock := range locks {
		if lock.Type == LockTypeExclusive {
			if lock.Owner != userID {
				return true, lock, fmt.Errorf("resource is locked exclusively by %s", lock.Owner)
			}
		}
	}

	return false, nil, nil
}

// CheckLockConflict 检查锁定冲突（用于创建新锁时）
func (lm *LockManager) CheckLockConflict(path string, newLockType LockType, userID string, depth int) (bool, *Lock, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// 检查精确路径匹配
	if locks, ok := lm.locksByPath[path]; ok {
		for _, lock := range locks {
			// 跳过过期的锁
			if time.Now().After(lock.ExpiresAt) {
				continue
			}

			// 排他锁与任何锁冲突
			if lock.Type == LockTypeExclusive || newLockType == LockTypeExclusive {
				if lock.Owner != userID {
					return true, lock, fmt.Errorf("conflicting lock exists")
				}
			}
		}
	}

	// 如果是深度锁，检查父路径
	if depth != 0 {
		if conflict, lock := lm.checkParentConflictsUnsafe(path, newLockType, userID); conflict {
			return true, lock, fmt.Errorf("parent path is locked")
		}
	}

	// 检查子路径（如果新锁是深度锁）
	if depth != 0 {
		if conflict, lock := lm.checkChildrenConflictsUnsafe(path, newLockType, userID); conflict {
			return true, lock, fmt.Errorf("child path is locked")
		}
	}

	return false, nil, nil
}

// checkParentConflictsUnsafe 检查父路径冲突（不加锁）
func (lm *LockManager) checkParentConflictsUnsafe(path string, newLockType LockType, userID string) (bool, *Lock) {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	for i := len(parts) - 1; i > 0; i-- {
		parentPath := "/" + strings.Join(parts[:i], "/")

		if locks, ok := lm.locksByPath[parentPath]; ok {
			for _, lock := range locks {
				// 跳过过期的锁
				if time.Now().After(lock.ExpiresAt) {
					continue
				}

				// 只有深度锁会影响子路径
				if lock.Depth != 0 {
					// 排他锁与任何锁冲突
					if lock.Type == LockTypeExclusive || newLockType == LockTypeExclusive {
						if lock.Owner != userID {
							return true, lock
						}
					}
				}
			}
		}
	}

	return false, nil
}

// checkChildrenConflictsUnsafe 检查子路径冲突（不加锁）
func (lm *LockManager) checkChildrenConflictsUnsafe(path string, newLockType LockType, userID string) (bool, *Lock) {
	prefix := strings.TrimSuffix(path, "/") + "/"

	for childPath, locks := range lm.locksByPath {
		// 检查是否为子路径
		if !strings.HasPrefix(childPath, prefix) && childPath != path {
			continue
		}

		for _, lock := range locks {
			// 跳过过期的锁
			if time.Now().After(lock.ExpiresAt) {
				continue
			}

			// 排他锁与任何锁冲突
			if lock.Type == LockTypeExclusive || newLockType == LockTypeExclusive {
				if lock.Owner != userID {
					return true, lock
				}
			}
		}
	}

	return false, nil
}

// CheckParentLocks 检查父目录锁定
func (lm *LockManager) CheckParentLocks(path string, userID string) (bool, *Lock, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	parts := strings.Split(strings.Trim(path, "/"), "/")

	for i := len(parts) - 1; i > 0; i-- {
		parentPath := "/" + strings.Join(parts[:i], "/")

		if locks, ok := lm.locksByPath[parentPath]; ok {
			for _, lock := range locks {
				// 跳过过期的锁
				if time.Now().After(lock.ExpiresAt) {
					continue
				}

				// 只有深度锁会影响子路径
				if lock.Depth != 0 {
					if lock.Type == LockTypeExclusive && lock.Owner != userID {
						return true, lock, fmt.Errorf("parent path is locked by %s", lock.Owner)
					}
				}
			}
		}
	}

	return false, nil, nil
}

// GetLockDiscovery 获取锁定发现信息
func (lm *LockManager) GetLockDiscovery(path string) []ActiveLock {
	locks := lm.GetLocksForPath(path)
	var activeLocks []ActiveLock

	for _, lock := range locks {
		activeLock := ActiveLock{
			LockScope: LockScopeInfo{},
			LockType: LockTypeInfo{
				Write: &struct{}{},
			},
			Depth:   fmt.Sprintf("%d", lock.Depth),
			Owner:   lock.Owner,
			Timeout: fmt.Sprintf("Second-%d", lock.Timeout),
			LockToken: LockToken{
				Href: lock.Token,
			},
			LockRoot: LockRoot{
				Href: lock.LockRoot,
			},
		}

		if lock.Type == LockTypeExclusive {
			activeLock.LockScope.Exclusive = &struct{}{}
		} else {
			activeLock.LockScope.Shared = &struct{}{}
		}

		// 注意：Depth已在上面正确设置为字符串
		activeLocks = append(activeLocks, activeLock)
	}

	return activeLocks
}

// CleanExpiredLocks 清理过期的锁定
func (lm *LockManager) CleanExpiredLocks() int {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	expiredTokens := []string{}

	for token, lock := range lm.locks {
		if now.After(lock.ExpiresAt) {
			expiredTokens = append(expiredTokens, token)
		}
	}

	for _, token := range expiredTokens {
		lm.removeLockUnsafe(token)
	}

	// 清理持久化存储中的过期锁定
	if lm.persistence != nil {
		if count, err := lm.persistence.CleanExpiredLocks(); err != nil {
			log.Printf("Warning: failed to clean expired locks from persistence: %v", err)
		} else if count > 0 {
			log.Printf("Cleaned %d expired locks from persistence", count)
		}
	}

	return len(expiredTokens)
}

// startCleanupTask 启动后台清理任务
func (lm *LockManager) startCleanupTask() {
	ticker := time.NewTicker(60 * time.Second) // 每60秒清理一次
	defer ticker.Stop()

	for range ticker.C {
		lm.CleanExpiredLocks()
	}
}

// GetAllLocks 获取所有活动锁定（用于调试和管理）
func (lm *LockManager) GetAllLocks() []*Lock {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	locks := make([]*Lock, 0, len(lm.locks))
	now := time.Now()

	for _, lock := range lm.locks {
		if now.Before(lock.ExpiresAt) {
			locks = append(locks, lock)
		}
	}

	return locks
}

// GetLockCount 获取活动锁定数量
func (lm *LockManager) GetLockCount() int {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	count := 0
	now := time.Now()

	for _, lock := range lm.locks {
		if now.Before(lock.ExpiresAt) {
			count++
		}
	}

	return count
}

// restoreFromPersistence 从持久化存储恢复锁定数据
func (lm *LockManager) restoreFromPersistence() error {
	if lm.persistence == nil {
		return nil
	}

	locks, err := lm.persistence.LoadAllLocks()
	if err != nil {
		return fmt.Errorf("failed to load locks from persistence: %v", err)
	}

	for _, lock := range locks {
		lm.locks[lock.Token] = lock
		lm.locksByPath[lock.Path] = append(lm.locksByPath[lock.Path], lock)
	}

	log.Printf("Restored %d locks from persistence", len(locks))
	return nil
}

// startSyncTask 启动同步任务
func (lm *LockManager) startSyncTask() {
	if lm.config == nil || lm.persistence == nil {
		return
	}

	ticker := time.NewTicker(lm.config.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		lm.performSync()
	}
}

// performSync 执行同步操作
func (lm *LockManager) performSync() {
	if lm.persistence == nil {
		return
	}

	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	
	// 同步内存中的锁定到持久化存储
	for token, lock := range lm.locks {
		if now.Before(lock.ExpiresAt) {
			if err := lm.persistence.SaveLock(lock); err != nil {
				log.Printf("Warning: failed to sync lock %s: %v", token, err)
			}
		}
	}

	lm.lastSync = now
}

// GetStatistics 获取锁定统计信息
func (lm *LockManager) GetStatistics() (*LockStats, error) {
	if lm.persistence != nil {
		return lm.persistence.GetStats()
	}

	// 如果没有持久化，返回内存统计
	stats := &LockStats{
		LocksByPath:  make(map[string]int),
		LocksByOwner: make(map[string]int),
	}

	lm.mu.RLock()
	defer lm.mu.RUnlock()

	now := time.Now()
	for _, lock := range lm.locks {
		if now.Before(lock.ExpiresAt) {
			stats.TotalLocks++
			stats.ActiveLocks++
			
			if lock.Type == LockTypeExclusive {
				stats.ExclusiveLocks++
			} else {
				stats.SharedLocks++
			}
			
			stats.LocksByPath[lock.Path]++
			stats.LocksByOwner[lock.Owner]++
			stats.AverageTimeout += float64(lock.Timeout)
		} else {
			stats.ExpiredLocks++
		}
	}

	if stats.TotalLocks > 0 {
		stats.AverageTimeout /= float64(stats.TotalLocks)
	}

	return stats, nil
}

// CreateBackup 创建锁定数据备份
func (lm *LockManager) CreateBackup(backupType, description string) (*BackupMetadata, error) {
	if lm.backup == nil {
		return nil, fmt.Errorf("backup manager not initialized")
	}

	return lm.backup.CreateBackup(backupType, description)
}

// RestoreBackup 从备份恢复锁定数据
func (lm *LockManager) RestoreBackup(backupPath string, options *RestoreOptions) error {
	if lm.backup == nil {
		return fmt.Errorf("backup manager not initialized")
	}

	return lm.backup.RestoreBackup(backupPath, options)
}

// ListBackups 列出所有备份
func (lm *LockManager) ListBackups() ([]*BackupMetadata, error) {
	if lm.backup == nil {
		return nil, fmt.Errorf("backup manager not initialized")
	}

	return lm.backup.ListBackups()
}

// VerifyBackup 验证备份完整性
func (lm *LockManager) VerifyBackup(backupPath string) error {
	if lm.backup == nil {
		return fmt.Errorf("backup manager not initialized")
	}

	return lm.backup.VerifyBackup(backupPath)
}

// DeleteBackup 删除备份
func (lm *LockManager) DeleteBackup(backupID int) error {
	if lm.backup == nil {
		return fmt.Errorf("backup manager not initialized")
	}

	return lm.backup.DeleteBackup(backupID)
}

// ForceSync 强制同步内存中的锁定到持久化存储
func (lm *LockManager) ForceSync() error {
	if lm.persistence == nil {
		return nil
	}

	lm.mu.Lock()
	defer lm.mu.Unlock()

	for token, lock := range lm.locks {
		if err := lm.persistence.SaveLock(lock); err != nil {
			return fmt.Errorf("failed to sync lock %s: %v", token, err)
		}
	}

	lm.lastSync = time.Now()
	log.Printf("Force synced %d locks to persistence", len(lm.locks))
	return nil
}

// GetLastSyncTime 获取最后同步时间
func (lm *LockManager) GetLastSyncTime() time.Time {
	return lm.lastSync
}

// Close 关闭锁定管理器
func (lm *LockManager) Close() error {
	// 强制同步
	if err := lm.ForceSync(); err != nil {
		log.Printf("Warning: failed to sync locks before close: %v", err)
	}

	// 关闭持久化管理器
	if lm.persistence != nil {
		if err := lm.persistence.Close(); err != nil {
			return fmt.Errorf("failed to close persistence: %v", err)
		}
	}

	return nil
}