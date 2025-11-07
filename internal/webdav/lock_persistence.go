package webdav

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/webdav-gateway/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

// LockPersistence 锁定持久化管理器
type LockPersistence struct {
	db         *sql.DB
	config     *config.LockPersistenceConfig
	mu         sync.RWMutex
	lastBackup time.Time
}

// LockData 持久化的锁定数据
type LockData struct {
	Token       string    `json:"token"`
	Type        string    `json:"type"`
	Scope       string    `json:"scope"`
	Owner       string    `json:"owner"`
	Timeout     int64     `json:"timeout"`
	CreatedAt   int64     `json:"created_at"`
	ExpiresAt   int64     `json:"expires_at"`
	Path        string    `json:"path"`
	Depth       int       `json:"depth"`
	LockRoot    string    `json:"lock_root"`
	RefreshHint int64     `json:"refresh_hint"`
	Version     int       `json:"version"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
}

// LockStats 锁定统计信息
type LockStats struct {
	TotalLocks         int            `json:"total_locks"`
	ExclusiveLocks     int            `json:"exclusive_locks"`
	SharedLocks        int            `json:"shared_locks"`
	ExpiredLocks       int            `json:"expired_locks"`
	ActiveLocks        int            `json:"active_locks"`
	AverageTimeout     float64        `json:"average_timeout"`
	LocksByPath        map[string]int `json:"locks_by_path"`
	LocksByOwner       map[string]int `json:"locks_by_owner"`
	LastCleanup        time.Time      `json:"last_cleanup"`
	BackupLastRun      time.Time      `json:"backup_last_run"`
}

// NewLockPersistence 创建新的持久化管理器
func NewLockPersistence(config *config.LockPersistenceConfig) (*LockPersistence, error) {
	if !config.Enabled {
		return nil, nil
	}

	// 确保存储目录存在
	if err := config.EnsureStorageDir(); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	// 打开或创建数据库
	db, err := sql.Open("sqlite3", config.StoragePath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000&_temp_store=MEMORY")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// 配置数据库连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	lp := &LockPersistence{
		db:     db,
		config: config,
	}

	// 初始化数据库表
	if err := lp.initDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	return lp, nil
}

// initDatabase 初始化数据库表结构
func (lp *LockPersistence) initDatabase() error {
	queries := []string{
		// 锁定数据表
		`CREATE TABLE IF NOT EXISTS locks (
			token TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			scope TEXT NOT NULL,
			owner TEXT NOT NULL,
			timeout INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			path TEXT NOT NULL,
			depth INTEGER NOT NULL,
			lock_root TEXT NOT NULL,
			refresh_hint INTEGER NOT NULL,
			version INTEGER DEFAULT 1,
			created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// 锁定历史表（用于审计）
		`CREATE TABLE IF NOT EXISTS lock_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token TEXT NOT NULL,
			action TEXT NOT NULL,
			user_id TEXT,
			path TEXT,
			details TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// 备份元数据表
		`CREATE TABLE IF NOT EXISTS backup_metadata (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			backup_type TEXT NOT NULL,
			backup_path TEXT NOT NULL,
			lock_count INTEGER NOT NULL,
			file_size INTEGER NOT NULL,
			checksum TEXT NOT NULL,
			created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// 统计信息表
		`CREATE TABLE IF NOT EXISTS lock_statistics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			stat_date DATE NOT NULL,
			total_locks INTEGER DEFAULT 0,
			exclusive_locks INTEGER DEFAULT 0,
			shared_locks INTEGER DEFAULT 0,
			expired_locks INTEGER DEFAULT 0,
			active_locks INTEGER DEFAULT 0,
			average_timeout REAL DEFAULT 0,
			created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// 创建索引
		`CREATE INDEX IF NOT EXISTS idx_locks_path ON locks(path)`,
		`CREATE INDEX IF NOT EXISTS idx_locks_owner ON locks(owner)`,
		`CREATE INDEX IF NOT EXISTS idx_locks_expires_at ON locks(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_lock_history_token ON lock_history(token)`,
		`CREATE INDEX IF NOT EXISTS idx_lock_history_timestamp ON lock_history(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_lock_statistics_date ON lock_statistics(stat_date)`,
	}

	for _, query := range queries {
		if _, err := lp.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
	}

	return nil
}

// SaveLock 保存锁定到持久化存储
func (lp *LockPersistence) SaveLock(lock *Lock) error {
	if lp == nil {
		return nil
	}

	lp.mu.Lock()
	defer lp.mu.Unlock()

	lockData := &LockData{
		Token:       lock.Token,
		Type:        string(lock.Type),
		Scope:       string(lock.Scope),
		Owner:       lock.Owner,
		Timeout:     lock.Timeout,
		CreatedAt:   lock.CreatedAt.Unix(),
		ExpiresAt:   lock.ExpiresAt.Unix(),
		Path:        lock.Path,
		Depth:       lock.Depth,
		LockRoot:    lock.LockRoot,
		RefreshHint: int64(lock.RefreshHint),
		Version:     1,
		Modified:    time.Now(),
	}

	data, err := json.Marshal(lockData)
	if err != nil {
		return fmt.Errorf("failed to marshal lock data: %v", err)
	}

	query := `
		INSERT OR REPLACE INTO locks 
		(token, type, scope, owner, timeout, created_at, expires_at, path, depth, lock_root, refresh_hint, version, modified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = lp.db.Exec(query,
		lockData.Token, lockData.Type, lockData.Scope, lockData.Owner,
		lockData.Timeout, lockData.CreatedAt, lockData.ExpiresAt,
		lockData.Path, lockData.Depth, lockData.LockRoot, lockData.RefreshHint,
		lockData.Version, lockData.Modified.Unix())

	if err != nil {
		return fmt.Errorf("failed to save lock: %v", err)
	}

	// 记录历史
	lp.recordHistory(lock.Token, "save", lock.Owner, lock.Path, string(data))

	return nil
}

// LoadLock 从持久化存储加载锁定
func (lp *LockPersistence) LoadLock(token string) (*Lock, error) {
	if lp == nil {
		return nil, nil
	}

	lp.mu.RLock()
	defer lp.mu.RUnlock()

	query := `SELECT * FROM locks WHERE token = ?`
	row := lp.db.QueryRow(query, token)

	var lockData LockData
	var created, modified int64

	err := row.Scan(
		&lockData.Token, &lockData.Type, &lockData.Scope, &lockData.Owner,
		&lockData.Timeout, &lockData.CreatedAt, &lockData.ExpiresAt,
		&lockData.Path, &lockData.Depth, &lockData.LockRoot, &lockData.RefreshHint,
		&lockData.Version, &created, &modified)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load lock: %v", err)
	}

	// 检查是否过期
	if time.Now().After(time.Unix(lockData.ExpiresAt, 0)) {
		// 标记为过期，但不立即删除
		return nil, nil
	}

	lock := &Lock{
		Token:       lockData.Token,
		Type:        LockType(lockData.Type),
		Scope:       LockScope(lockData.Scope),
		Owner:       lockData.Owner,
		Timeout:     lockData.Timeout,
		CreatedAt:   time.Unix(lockData.CreatedAt, 0),
		ExpiresAt:   time.Unix(lockData.ExpiresAt, 0),
		Path:        lockData.Path,
		Depth:       lockData.Depth,
		LockRoot:    lockData.LockRoot,
		RefreshHint: time.Duration(lockData.RefreshHint),
	}

	return lock, nil
}

// LoadAllLocks 加载所有有效锁定
func (lp *LockPersistence) LoadAllLocks() ([]*Lock, error) {
	if lp == nil {
		return nil, nil
	}

	lp.mu.RLock()
	defer lp.mu.RUnlock()

	query := `SELECT * FROM locks WHERE expires_at > ? ORDER BY created_at`
	rows, err := lp.db.Query(query, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to query locks: %v", err)
	}
	defer rows.Close()

	var locks []*Lock
	for rows.Next() {
		var lockData LockData
		var created, modified int64

		err := rows.Scan(
			&lockData.Token, &lockData.Type, &lockData.Scope, &lockData.Owner,
			&lockData.Timeout, &lockData.CreatedAt, &lockData.ExpiresAt,
			&lockData.Path, &lockData.Depth, &lockData.LockRoot, &lockData.RefreshHint,
			&lockData.Version, &created, &modified)

		if err != nil {
			continue // 跳过损坏的记录
		}

		// 检查是否过期
		if time.Now().After(time.Unix(lockData.ExpiresAt, 0)) {
			continue
		}

		lock := &Lock{
			Token:       lockData.Token,
			Type:        LockType(lockData.Type),
			Scope:       LockScope(lockData.Scope),
			Owner:       lockData.Owner,
			Timeout:     lockData.Timeout,
			CreatedAt:   time.Unix(lockData.CreatedAt, 0),
			ExpiresAt:   time.Unix(lockData.ExpiresAt, 0),
			Path:        lockData.Path,
			Depth:       lockData.Depth,
			LockRoot:    lockData.LockRoot,
			RefreshHint: time.Duration(lockData.RefreshHint),
		}

		locks = append(locks, lock)
	}

	return locks, nil
}

// DeleteLock 从持久化存储删除锁定
func (lp *LockPersistence) DeleteLock(token string) error {
	if lp == nil {
		return nil
	}

	lp.mu.Lock()
	defer lp.mu.Unlock()

	query := `DELETE FROM locks WHERE token = ?`
	_, err := lp.db.Exec(query, token)
	if err != nil {
		return fmt.Errorf("failed to delete lock: %v", err)
	}

	// 记录历史
	lp.recordHistory(token, "delete", "", "", "")

	return nil
}

// CleanExpiredLocks 清理过期的锁定
func (lp *LockPersistence) CleanExpiredLocks() (int, error) {
	if lp == nil {
		return 0, nil
	}

	lp.mu.Lock()
	defer lp.mu.Unlock()

	now := time.Now().Unix()
	query := `SELECT token FROM locks WHERE expires_at <= ?`
	rows, err := lp.db.Query(query, now)
	if err != nil {
		return 0, fmt.Errorf("failed to query expired locks: %v", err)
	}
	defer rows.Close()

	var expiredTokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err == nil {
			expiredTokens = append(expiredTokens, token)
		}
	}

	// 删除过期锁定
	deleteQuery := `DELETE FROM locks WHERE token = ?`
	for _, token := range expiredTokens {
		if _, err := lp.db.Exec(deleteQuery, token); err == nil {
			lp.recordHistory(token, "expire", "", "", "")
		}
	}

	return len(expiredTokens), nil
}

// GetStats 获取锁定统计信息
func (lp *LockPersistence) GetStats() (*LockStats, error) {
	if lp == nil {
		return nil, nil
	}

	lp.mu.RLock()
	defer lp.mu.RUnlock()

	stats := &LockStats{
		LocksByPath:  make(map[string]int),
		LocksByOwner: make(map[string]int),
	}

	// 获取基本统计
	query := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN type = 'exclusive' THEN 1 ELSE 0 END) as exclusive,
			SUM(CASE WHEN type = 'shared' THEN 1 ELSE 0 END) as shared,
			SUM(CASE WHEN expires_at <= ? THEN 1 ELSE 0 END) as expired,
			AVG(timeout) as avg_timeout
		FROM locks
	`
	
	row := lp.db.QueryRow(query, time.Now().Unix())
	
	err := row.Scan(
		&stats.TotalLocks,
		&stats.ExclusiveLocks,
		&stats.SharedLocks,
		&stats.ExpiredLocks,
		&stats.AverageTimeout)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	stats.ActiveLocks = stats.TotalLocks - stats.ExpiredLocks

	// 获取按路径分布
	pathQuery := `SELECT path, COUNT(*) FROM locks WHERE expires_at > ? GROUP BY path`
	pathRows, err := lp.db.Query(pathQuery, time.Now().Unix())
	if err == nil {
		defer pathRows.Close()
		for pathRows.Next() {
			var path string
			var count int
			if err := pathRows.Scan(&path, &count); err == nil {
				stats.LocksByPath[path] = count
			}
		}
	}

	// 获取按所有者分布
	ownerQuery := `SELECT owner, COUNT(*) FROM locks WHERE expires_at > ? GROUP BY owner`
	ownerRows, err := lp.db.Query(ownerQuery, time.Now().Unix())
	if err == nil {
		defer ownerRows.Close()
		for ownerRows.Next() {
			var owner string
			var count int
			if err := ownerRows.Scan(&owner, &count); err == nil {
				stats.LocksByOwner[owner] = count
			}
		}
	}

	return stats, nil
}

// recordHistory 记录锁定历史
func (lp *LockPersistence) recordHistory(token, action, userID, path, details string) {
	query := `INSERT INTO lock_history (token, action, user_id, path, details) VALUES (?, ?, ?, ?, ?)`
	_, _ = lp.db.Exec(query, token, action, userID, path, details)
}

// Close 关闭持久化管理器
func (lp *LockPersistence) Close() error {
	if lp == nil || lp.db == nil {
		return nil
	}
	return lp.db.Close()
}

// GetBackupPath 获取备份文件路径
func (lp *LockPersistence) GetBackupPath(backupType string) string {
	if lp == nil {
		return ""
	}

	dir := filepath.Dir(lp.config.StoragePath)
	timestamp := time.Now().Format("20060102_150405")
	return filepath.Join(dir, fmt.Sprintf("locks_backup_%s_%s.json", backupType, timestamp))
}