-- WebDAV锁定表初始化脚本
-- 用于创建锁定持久化存储所需的表和索引

-- 创建锁定表
CREATE TABLE IF NOT EXISTS webdav_locks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token VARCHAR(255) UNIQUE NOT NULL,
    path TEXT NOT NULL,
    user_id UUID NOT NULL,
    lock_type VARCHAR(20) NOT NULL CHECK (lock_type IN ('EXCLUSIVE', 'SHARED')),
    depth INTEGER NOT NULL DEFAULT 0,
    owner TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_webdav_locks_path ON webdav_locks (path);
CREATE INDEX IF NOT EXISTS idx_webdav_locks_expires ON webdav_locks (expires_at);
CREATE INDEX IF NOT EXISTS idx_webdav_locks_user ON webdav_locks (user_id);
CREATE INDEX IF NOT EXISTS idx_webdav_locks_token ON webdav_locks (token);
CREATE INDEX IF NOT EXISTS idx_webdav_locks_active ON webdav_locks (path) WHERE expires_at > NOW();

-- 创建复合索引用于常用查询
CREATE INDEX IF NOT EXISTS idx_webdav_locks_path_active ON webdav_locks (path, lock_type) WHERE expires_at > NOW();
CREATE INDEX IF NOT EXISTS idx_webdav_locks_user_active ON webdav_locks (user_id, expires_at) WHERE expires_at > NOW();

-- 创建锁定统计视图
CREATE OR REPLACE VIEW webdav_lock_stats AS
SELECT 
    COUNT(*) as total_locks,
    COUNT(*) FILTER (WHERE expires_at > NOW()) as active_locks,
    COUNT(*) FILTER (WHERE lock_type = 'EXCLUSIVE') as exclusive_locks,
    COUNT(*) FILTER (WHERE lock_type = 'SHARED') as shared_locks,
    COUNT(*) FILTER (WHERE expires_at <= NOW()) as expired_locks,
    AVG(EXTRACT(EPOCH FROM (expires_at - created_at))) as avg_lock_duration,
    MAX(EXTRACT(EPOCH FROM (expires_at - created_at))) as max_lock_duration
FROM webdav_locks;

-- 创建锁定清理函数
CREATE OR REPLACE FUNCTION cleanup_expired_locks()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM webdav_locks 
    WHERE expires_at < NOW() - INTERVAL '1 hour';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- 创建锁定清理存储过程
CREATE OR REPLACE FUNCTION cleanup_locks_batch(batch_size INTEGER DEFAULT 1000)
RETURNS TABLE(cleaned_count INTEGER, remaining_count INTEGER) AS $$
DECLARE
    cleaned INTEGER := 0;
    remaining INTEGER := 0;
BEGIN
    -- 清理过期锁定
    WITH expired_locks AS (
        SELECT id 
        FROM webdav_locks 
        WHERE expires_at < NOW() - INTERVAL '1 hour'
        ORDER BY expires_at ASC
        LIMIT batch_size
    )
    DELETE FROM webdav_locks 
    WHERE id IN (SELECT id FROM expired_locks);
    
    GET DIAGNOSTICS cleaned = ROW_COUNT;
    
    -- 计算剩余锁定数量
    SELECT COUNT(*) INTO remaining
    FROM webdav_locks 
    WHERE expires_at > NOW();
    
    RETURN QUERY SELECT cleaned, remaining;
END;
$$ LANGUAGE plpgsql;

-- 创建锁定冲突检测函数
CREATE OR REPLACE FUNCTION check_lock_conflicts(
    p_path TEXT,
    p_user_id UUID,
    p_lock_type VARCHAR(20)
)
RETURNS TABLE(
    has_conflict BOOLEAN,
    conflicting_locks JSON
) AS $$
DECLARE
    conflicts JSON;
    conflict_exists BOOLEAN := FALSE;
BEGIN
    -- 检查EXCLUSIVE锁冲突
    IF p_lock_type = 'EXCLUSIVE' THEN
        SELECT COALESCE(
            json_agg(
                json_build_object(
                    'token', token,
                    'user_id', user_id,
                    'lock_type', lock_type,
                    'owner', owner,
                    'expires_at', expires_at
                )
            ), 
            '[]'::json
        ) INTO conflicts
        FROM webdav_locks
        WHERE path = p_path 
        AND expires_at > NOW();
        
        IF conflicts != '[]'::json THEN
            conflict_exists := TRUE;
        END IF;
    
    -- 检查SHARED锁冲突
    ELSIF p_lock_type = 'SHARED' THEN
        SELECT COALESCE(
            json_agg(
                json_build_object(
                    'token', token,
                    'user_id', user_id,
                    'lock_type', lock_type,
                    'owner', owner,
                    'expires_at', expires_at
                )
            ),
            '[]'::json
        ) INTO conflicts
        FROM webdav_locks
        WHERE path = p_path 
        AND lock_type = 'EXCLUSIVE'
        AND expires_at > NOW();
        
        IF conflicts != '[]'::json THEN
            conflict_exists := TRUE;
        END IF;
    END IF;
    
    RETURN QUERY SELECT conflict_exists, conflicts;
END;
$$ LANGUAGE plpgsql;

-- 创建锁定统计函数
CREATE OR REPLACE FUNCTION get_lock_statistics()
RETURNS TABLE(
    total_locks BIGINT,
    active_locks BIGINT,
    exclusive_locks BIGINT,
    shared_locks BIGINT,
    expired_locks BIGINT,
    avg_duration NUMERIC,
    max_duration NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_locks,
        COUNT(*) FILTER (WHERE expires_at > NOW()) as active_locks,
        COUNT(*) FILTER (WHERE lock_type = 'EXCLUSIVE') as exclusive_locks,
        COUNT(*) FILTER (WHERE lock_type = 'SHARED') as shared_locks,
        COUNT(*) FILTER (WHERE expires_at <= NOW()) as expired_locks,
        AVG(EXTRACT(EPOCH FROM (expires_at - created_at))) as avg_duration,
        MAX(EXTRACT(EPOCH FROM (expires_at - created_at))) as max_duration
    FROM webdav_locks;
END;
$$ LANGUAGE plpgsql;

-- 创建定期清理任务（需要pg_cron扩展）
-- 取消注释以下行以启用自动清理（需要安装pg_cron扩展）
-- SELECT cron.schedule('cleanup-expired-locks', '*/5 * * * *', 'SELECT cleanup_expired_locks();');

-- 创建触发器以自动更新updated_at字段
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_webdav_locks_updated_at
    BEFORE UPDATE ON webdav_locks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 插入示例数据（仅用于测试）
-- INSERT INTO webdav_locks (token, path, user_id, lock_type, owner, expires_at)
-- VALUES 
--     ('opaquelocktoken:test-token-1', '/test/file1.txt', '00000000-0000-0000-0000-000000000001', 'EXCLUSIVE', 'test@example.com', NOW() + INTERVAL '1 hour'),
--     ('opaquelocktoken:test-token-2', '/test/file2.txt', '00000000-0000-0000-0000-000000000002', 'SHARED', 'user@example.com', NOW() + INTERVAL '2 hours');

-- 创建权限（如果需要）
-- GRANT SELECT, INSERT, UPDATE, DELETE ON webdav_locks TO webdav;
-- GRANT USAGE ON SCHEMA public TO webdav;

-- 注释：生产环境中建议创建专用用户和权限
-- CREATE USER webdav_locks WITH PASSWORD 'secure_password';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON webdav_locks TO webdav_locks;
-- GRANT USAGE ON SCHEMA public TO webdav_locks;

-- 输出初始化完成信息
DO $$
BEGIN
    RAISE NOTICE 'WebDAV锁定表初始化完成';
    RAISE NOTICE '已创建表: webdav_locks';
    RAISE NOTICE '已创建索引: 6个';
    RAISE NOTICE '已创建函数: 4个';
    RAISE NOTICE '已创建视图: 1个';
    RAISE NOTICE '已创建触发器: 1个';
END $$;