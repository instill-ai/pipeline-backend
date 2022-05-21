package config

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/internal/logger"
)

// Config - Global variable to export
var Config AppConfig

// AppConfig defines
type AppConfig struct {
	Server           ServerConfig           `koanf:"server"`
	Database         DatabaseConfig         `koanf:"database"`
	Temporal         TemporalConfig         `koanf:"temporal"`
	Cache            CacheConfig            `koanf:"cache"`
	MgmtBackend      MgmtBackendConfig      `koanf:"mgmtbackend"`
	ConnectorBackend ConnectorBackendConfig `koanf:"connectorbackend"`
	ModelBackend     ModelBackendConfig     `koanf:"modelbackend"`
}

// ServerConfig defines HTTP server configurations
type ServerConfig struct {
	Port  int `koanf:"port"`
	HTTPS struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
	CORSOrigins []string `koanf:"corsorigins"`
	Paginate    struct {
		Salt string `koanf:"salt"`
	}
}

// DatabaseConfig related to database
type DatabaseConfig struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Name     string `koanf:"name"`
	Version  uint   `koanf:"version"`
	TimeZone string `koanf:"timezone"`
	Pool     struct {
		IdleConnections int           `koanf:"idleconnections"`
		MaxConnections  int           `koanf:"maxconnections"`
		ConnLifeTime    time.Duration `koanf:"connlifetime"`
	}
}

// TemporalConfig related to Temporal
type TemporalConfig struct {
	ClientOptions client.Options `koanf:"clientoptions"`
}

// CacheConfig related to Redis
type CacheConfig struct {
	Redis struct {
		RedisOptions redis.Options `koanf:"redisoptions"`
	}
}

// MgmtBackendConfig related to mgmt-backend
type MgmtBackendConfig struct {
	Host  string `koanf:"host"`
	Port  int    `koanf:"port"`
	HTTPS struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// ConnectorBackendConfig related to connector-backend
type ConnectorBackendConfig struct {
	Host  string `koanf:"host"`
	Port  int    `koanf:"port"`
	HTTPS struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// ModelBackendConfig related to model-backend
type ModelBackendConfig struct {
	Host  string `koanf:"host"`
	Port  int    `koanf:"port"`
	HTTPS struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// Init - Assign global config to decoded config struct
func Init() error {
	logger, _ := logger.GetZapLogger()

	k := koanf.New(".")
	parser := yaml.Parser()

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fileRelativePath := fs.String("file", "config/config.yaml", "configuration file")
	flag.Parse()

	if err := k.Load(file.Provider(*fileRelativePath), parser); err != nil {
		logger.Fatal(err.Error())
	}

	if err := k.Load(env.Provider("CFG_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "CFG_")), "_", ".", -1)
	}), nil); err != nil {
		return err
	}

	if err := k.Unmarshal("", &Config); err != nil {
		return err
	}

	return ValidateConfig(&Config)
}

// ValidateConfig is for custom validation rules for the configuration
func ValidateConfig(cfg *AppConfig) error {
	return nil
}
