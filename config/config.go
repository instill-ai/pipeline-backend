package config

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/redis/go-redis/v9"

	"github.com/instill-ai/x/minio"
	"github.com/instill-ai/x/temporal"
)

const (
	EditionLocalCE      = "local-ce:dev"
	EditionCloudDev     = "cloud:dev"
	EditionCloudStaging = "cloud:staging"
	EditionCloudProd    = "cloud"
)

// Config - Global variable to export
var Config AppConfig

// AppConfig defines
type AppConfig struct {
	Server          ServerConfig          `koanf:"server"`
	Component       ComponentConfig       `koanf:"component"`
	Database        DatabaseConfig        `koanf:"database"`
	InfluxDB        InfluxDBConfig        `koanf:"influxdb"`
	Temporal        temporal.ClientConfig `koanf:"temporal"`
	Cache           CacheConfig           `koanf:"cache"`
	Log             LogConfig             `koanf:"log"`
	MgmtBackend     MgmtBackendConfig     `koanf:"mgmtbackend"`
	ModelBackend    ModelBackendConfig    `koanf:"modelbackend"`
	OpenFGA         OpenFGAConfig         `koanf:"openfga"`
	ArtifactBackend ArtifactBackendConfig `koanf:"artifactbackend"`
	Minio           minio.Config          `koanf:"minio"`
	AgentBackend    AgentBackendConfig    `koanf:"agentbackend"`
	APIGateway      APIGatewayConfig      `koanf:"apigateway"`
}

// APIGateway config
type APIGatewayConfig struct {
	Host       string `koanf:"host"`
	PublicPort int    `koanf:"publicport"`
	TLSEnabled bool   `koanf:"tlsenabled"`
}

// OpenFGA config
type OpenFGAConfig struct {
	Host    string `koanf:"host"`
	Port    int    `koanf:"port"`
	Replica struct {
		Host                 string `koanf:"host"`
		Port                 int    `koanf:"port"`
		ReplicationTimeFrame int    `koanf:"replicationtimeframe"` // in seconds
	} `koanf:"replica"`
}

// ServerConfig defines HTTP server configurations
type ServerConfig struct {
	PublicPort  int `koanf:"publicport"`
	PrivatePort int `koanf:"privateport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
	Edition string `koanf:"edition"`
	Usage   struct {
		Enabled    bool   `koanf:"enabled"`
		TLSEnabled bool   `koanf:"tlsenabled"`
		Host       string `koanf:"host"`
		Port       int    `koanf:"port"`
	}
	Debug       bool `koanf:"debug"`
	MaxDataSize int  `koanf:"maxdatasize"`
	Workflow    struct {
		MaxWorkflowTimeout int32 `koanf:"maxworkflowtimeout"`
		MaxWorkflowRetry   int32 `koanf:"maxworkflowretry"`
		MaxActivityRetry   int32 `koanf:"maxactivityretry"`
	}
	InstanceID      string `koanf:"instanceid"`
	InstillCoreHost string `koanf:"instillcorehost"`
}

// ComponentConfig contains the configuration of different components. Global
// secrets may be defined here by component, allowing them to have e.g. a
// default API key when no setup is specified, or to connect with a 3rd party
// vendor via OAuth.
type ComponentConfig struct {
	Secrets                     ComponentSecrets
	InternalUserEmails          []string
	ComponentsWithInternalUsers []string
}

// ComponentSecrets contains the global config secrets of each
// implemented component (referenced by ID). Components may use these secrets
// to skip the component configuration step and have a ready-to-run
// config.
type ComponentSecrets map[string]map[string]any

// DatabaseConfig related to database
type DatabaseConfig struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Replica  struct {
		Username             string `koanf:"username"`
		Password             string `koanf:"password"`
		Host                 string `koanf:"host"`
		Port                 int    `koanf:"port"`
		ReplicationTimeFrame int    `koanf:"replicationtimeframe"` // in seconds
	} `koanf:"replica"`
	Name     string `koanf:"name"`
	Version  uint   `koanf:"version"`
	TimeZone string `koanf:"timezone"`
	Pool     struct {
		IdleConnections int           `koanf:"idleconnections"`
		MaxConnections  int           `koanf:"maxconnections"`
		ConnLifeTime    time.Duration `koanf:"connlifetime"`
	}
}

// InfluxDBConfig related to influxDB database
type InfluxDBConfig struct {
	URL           string        `koanf:"url"`
	Token         string        `koanf:"token"`
	Org           string        `koanf:"org"`
	Bucket        string        `koanf:"bucket"`
	FlushInterval time.Duration `koanf:"flushinterval"`
	HTTPS         struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// LogConfig related to logging
type LogConfig struct {
	External      bool `koanf:"external"`
	OtelCollector struct {
		Host string `koanf:"host"`
		Port string `koanf:"port"`
	}
}

// CacheConfig related to Redis
type CacheConfig struct {
	Redis struct {
		RedisOptions redis.Options `koanf:"redisoptions"`
	}
}

// MgmtBackendConfig related to mgmt-backend
type MgmtBackendConfig struct {
	Host        string `koanf:"host"`
	PublicPort  int    `koanf:"publicport"`
	PrivatePort int    `koanf:"privateport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// ModelBackendConfig related to mgmt-backend
type ModelBackendConfig struct {
	Host       string `koanf:"host"`
	PublicPort int    `koanf:"publicport"`
	HTTPS      struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

type ArtifactBackendConfig struct {
	Host        string `koanf:"host"`
	PublicPort  int    `koanf:"publicport"`
	PrivatePort int    `koanf:"privateport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

type AgentBackendConfig struct {
	Host       string `koanf:"host"`
	PublicPort int    `koanf:"publicport"`
	HTTPS      struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// Init - Assign global config to decoded config struct
func Init(filePath string) error {
	k := koanf.New(".")
	parser := yaml.Parser()

	if err := k.Load(confmap.Provider(map[string]interface{}{
		"database.replica.replicationtimeframe": 60,
		"openfga.replica.replicationtimeframe":  60,
	}, "."), nil); err != nil {
		log.Fatal(err.Error())
	}

	if err := k.Load(file.Provider(filePath), parser); err != nil {
		log.Fatal(err.Error())
	}

	if err := k.Load(env.ProviderWithValue("CFG_", ".", func(s string, v string) (string, interface{}) {
		key := strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "CFG_")), "_", ".")
		if strings.Contains(v, ",") {
			return key, strings.Split(strings.TrimSpace(v), ",")
		}
		return key, v
	}), nil); err != nil {
		return err
	}

	if err := k.Unmarshal("", &Config); err != nil {
		return err
	}

	return ValidateConfig(&Config)
}

// ValidateConfig is for custom validation rules for the configuration
func ValidateConfig(_ *AppConfig) error {
	return nil
}

var defaultConfigPath = "config/config.yaml"

// ParseConfigFlag allows clients to specify the relative path to the file from
// which the configuration will be loaded.
func ParseConfigFlag() string {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	configPath := fs.String("file", defaultConfigPath, "configuration file")
	flag.Parse()

	return *configPath
}
