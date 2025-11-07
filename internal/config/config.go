package config

import (
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Database DatabaseConfig `mapstructure:"database"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Address     string        `mapstructure:"address"`
	Mode        string        `mapstructure:"mode"`
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret     string        `mapstructure:"jwt_secret"`
	TokenExpiry   time.Duration `mapstructure:"token_expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type     string            `mapstructure:"type"`
	MinIO    MinIOConfig       `mapstructure:"minio"`
	Local    LocalConfig       `mapstructure:"local"`
	Metadata map[string]string `mapstructure:"metadata"`
}

// MinIOConfig MinIO配置
type MinIOConfig struct {
	Endpoint   string `mapstructure:"endpoint"`
	AccessKey  string `mapstructure:"access_key"`
	SecretKey  string `mapstructure:"secret_key"`
	UseSSL     bool   `mapstructure:"use_ssl"`
	BucketName string `mapstructure:"bucket_name"`
	BucketPrefix string `mapstructure:"bucket_prefix"`
}

// LocalConfig 本地存储配置
type LocalConfig struct {
	RootPath string `mapstructure:"root_path"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string            `mapstructure:"type"`
	Postgres PostgresConfig    `mapstructure:"postgres"`
	SQLite   SQLiteConfig      `mapstructure:"sqlite"`
	Metadata map[string]string `mapstructure:"metadata"`
}

// PostgresConfig PostgreSQL配置
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// SQLiteConfig SQLite配置
type SQLiteConfig struct {
	Path string `mapstructure:"path"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type     string      `mapstructure:"type"`
	Redis    RedisConfig `mapstructure:"redis"`
	Memory   MemoryConfig `mapstructure:"memory"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Address  string        `mapstructure:"address"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// MemoryConfig 内存缓存配置
type MemoryConfig struct {
	DefaultExpiration time.Duration `mapstructure:"default_expiration"`
	GCInterval        time.Duration `mapstructure:"gc_interval"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Load 加载配置
func Load() (*Config, error) {
	// 设置默认值
	viper.SetDefault("server.address", ":8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", 10*time.Second)
	viper.SetDefault("server.write_timeout", 10*time.Second)
	viper.SetDefault("auth.jwt_secret", "your-secret-key")
	viper.SetDefault("auth.token_expiry", 24*time.Hour)
	viper.SetDefault("auth.refresh_expiry", 7*24*time.Hour)
	viper.SetDefault("storage.type", "minio")
	viper.SetDefault("storage.minio.endpoint", "localhost:9000")
	viper.SetDefault("storage.minio.use_ssl", false)
	viper.SetDefault("storage.minio.bucket_name", "webdav-files")
	viper.SetDefault("storage.minio.bucket_prefix", "user-")
	viper.SetDefault("storage.local.root_path", "./data")
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.sqlite.path", "./data/app.db")
	viper.SetDefault("cache.type", "memory")
	viper.SetDefault("cache.memory.default_expiration", 1*time.Hour)
	viper.SetDefault("cache.memory.gc_interval", 10*time.Minute)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")

	// 优先从配置文件加载
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/webdav-gateway")
	viper.AddConfigPath("$HOME/.webdav-gateway")

	// 从环境变量加载
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err == nil {
		// 配置文件读取成功
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		// 配置文件未找到但有其他错误
	}

	// 如果设置了环境变量，覆盖配置文件
	setEnvOverrides()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setEnvOverrides 设置环境变量覆盖
func setEnvOverrides() {
	// 服务器配置
	if addr := os.Getenv("SERVER_ADDRESS"); addr != "" {
		viper.Set("server.address", addr)
	}
	if mode := os.Getenv("SERVER_MODE"); mode != "" {
		viper.Set("server.mode", mode)
	}

	// 认证配置
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		viper.Set("auth.jwt_secret", secret)
	}

	// MinIO配置
	if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		viper.Set("storage.minio.endpoint", endpoint)
	}
	if accessKey := os.Getenv("MINIO_ACCESS_KEY"); accessKey != "" {
		viper.Set("storage.minio.access_key", accessKey)
	}
	if secretKey := os.Getenv("MINIO_SECRET_KEY"); secretKey != "" {
		viper.Set("storage.minio.secret_key", secretKey)
	}
	if bucket := os.Getenv("MINIO_BUCKET_NAME"); bucket != "" {
		viper.Set("storage.minio.bucket_name", bucket)
	}
	if bucketPrefix := os.Getenv("MINIO_BUCKET_PREFIX"); bucketPrefix != "" {
		viper.Set("storage.minio.bucket_prefix", bucketPrefix)
	}

	// PostgreSQL配置
	if pgHost := os.Getenv("POSTGRES_HOST"); pgHost != "" {
		viper.Set("database.postgres.host", pgHost)
	}
	if pgPort := os.Getenv("POSTGRES_PORT"); pgPort != "" {
		if port, err := strconv.Atoi(pgPort); err == nil {
			viper.Set("database.postgres.port", port)
		}
	}
	if pgUser := os.Getenv("POSTGRES_USERNAME"); pgUser != "" {
		viper.Set("database.postgres.username", pgUser)
	}
	if pgPassword := os.Getenv("POSTGRES_PASSWORD"); pgPassword != "" {
		viper.Set("database.postgres.password", pgPassword)
	}
	if pgDatabase := os.Getenv("POSTGRES_DATABASE"); pgDatabase != "" {
		viper.Set("database.postgres.database", pgDatabase)
	}

	// Redis配置
	if redisAddr := os.Getenv("REDIS_ADDRESS"); redisAddr != "" {
		viper.Set("cache.redis.address", redisAddr)
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		viper.Set("cache.redis.password", redisPassword)
	}
	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		if db, err := strconv.Atoi(redisDB); err == nil {
			viper.Set("cache.redis.db", db)
		}
	}
}

// GetDSN 获取数据库连接字符串
func (c *Config) GetDSN() string {
	switch c.Database.Type {
	case "postgres":
		return buildPostgresDSN(c.Database.Postgres)
	case "sqlite":
		return c.Database.SQLite.Path
	default:
		return ""
	}
}

// buildPostgresDSN 构建PostgreSQL DSN
func buildPostgresDSN(config PostgresConfig) string {
	dsn := "host=" + config.Host
	dsn += " port=" + strconv.Itoa(config.Port)
	dsn += " user=" + config.Username
	dsn += " password=" + config.Password
	dsn += " dbname=" + config.Database
	dsn += " sslmode=" + config.SSLMode
	return dsn
}

// IsProduction 检查是否为生产环境
func (c *Config) IsProduction() bool {
	return c.Server.Mode == "production" || c.Server.Mode == "release"
}

// GetGINMode 获取Gin模式
func (c *Config) GetGINMode() string {
	switch c.Server.Mode {
	case "debug":
		return gin.DebugMode
	case "release", "production":
		return gin.ReleaseMode
	case "test":
		return gin.TestMode
	default:
		return gin.DebugMode
	}
}