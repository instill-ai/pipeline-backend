package sql

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/jmoiron/sqlx"
	"google.golang.org/protobuf/types/known/structpb"

	mysql "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/nakagami/firebirdsql"
	_ "github.com/sijms/go-ora"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var enginesMTLS = map[string]string{
	"PostgreSQL": "postgresql://%s:%s@%s/%s?sslmode=verify-full&sslrootcert=%s&sslcert=%s&sslkey=%s",          // PostgreSQL
	"SQL Server": "sqlserver://%s:%s@%s?database=%s&encrypt=true&trustServerCertificate=false&certificate=%s", // SQL Server (mTLS not supported directly in DSN)
	"Oracle":     "oracle://%s:%s@%s/%s?ssl=true&sslrootcert=%s&sslcert=%s&sslkey=%s",                         // Oracle
	"MySQL":      "%s:%s@tcp(%s)/%s?tls=custom",                                                               // MySQL and MariaDB
	"Firebird":   "firebirdsql://%s:%s@%s/%s?sslmode=verify-full&sslrootcert=%s&sslcert=%s&sslkey=%s",         // Firebird
}

var enginesTLS = map[string]string{
	"PostgreSQL": "postgresql://%s:%s@%s/%s?sslmode=verify-full&sslrootcert=%s",                               // PostgreSQL
	"SQL Server": "sqlserver://%s:%s@%s?database=%s&encrypt=true&trustServerCertificate=false&certificate=%s", // SQL Server
	"Oracle":     "oracle://%s:%s@%s/%s?ssl=true&sslrootcert=%s",                                              // Oracle
	"MySQL":      "%s:%s@tcp(%s)/%s?tls=custom",                                                               // MySQL and MariaDB
	"Firebird":   "firebirdsql://%s:%s@%s/%s?sslmode=verify-full&sslrootcert=%s",                              // Firebird
}

var engines = map[string]string{
	"PostgreSQL": "postgresql://%s:%s@%s/%s",                                                  // PostgreSQL
	"SQL Server": "sqlserver://%s:%s@%s?database=%s&encrypt=true&trustServerCertificate=true", // SQL Server
	"Oracle":     "oracle://%s:%s@%s/%s",                                                      // Oracle
	"MySQL":      "%s:%s@tcp(%s)/%s",                                                          // MySQL and MariaDB
	"Firebird":   "firebirdsql://%s:%s@%s/%s",                                                 // Firebird
}

var enginesType = map[string]string{
	"PostgreSQL": "postgres",    // PostgreSQL
	"SQL Server": "sqlserver",   // SQL Server
	"Oracle":     "oracle",      // Oracle
	"MySQL":      "mysql",       // MySQL and MariaDB
	"Firebird":   "firebirdsql", // Firebird
}

type Config struct {
	DBEngine     string
	DBUsername   string
	DBPassword   string
	DBName       string
	DBHost       string
	DBPort       string
	DBSSLTLSType string
}

type SSLTLSConfig struct {
	CA   string `json:"ssl-tls-ca"`
	Cert string `json:"ssl-tls-cert"`
	Key  string `json:"ssl-tls-key"`
}

func LoadConfig(setup *structpb.Struct, ssltls *structpb.Struct) *Config {
	return &Config{
		DBEngine:     getEngine(setup),
		DBUsername:   getUsername(setup),
		DBPassword:   getPassword(setup),
		DBName:       getDatabaseName(setup),
		DBHost:       getHost(setup),
		DBPort:       getPort(setup),
		DBSSLTLSType: getSSLTLSType(ssltls),
	}
}

func createTempFile(decodedBase64 []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "*.pem")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(decodedBase64); err != nil {
		return "", fmt.Errorf("failed to write data to temporary file: %w", err)
	}

	return tmpFile.Name(), nil
}

func tlsConfig(caFilePath string, certFilePath string, keyFilePath string) (*tls.Config, error) {
	rootCertPool := x509.NewCertPool()
	pem, err := os.ReadFile(caFilePath)
	if err != nil {
		return nil, err
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return nil, fmt.Errorf("failed to append PEM")
	}
	if certFilePath == "" && keyFilePath == "" {
		return &tls.Config{
			RootCAs: rootCertPool,
		}, nil
	} else {
		cert, err := tls.LoadX509KeyPair(certFilePath, keyFilePath)
		if err != nil {
			log.Fatal(err)
		}
		return &tls.Config{
			RootCAs:      rootCertPool,
			Certificates: []tls.Certificate{cert},
		}, nil
	}
}

func newClient(setup *structpb.Struct) (SQLClient, error) {
	ssltls := setup.GetFields()["ssl-tls"].GetStructValue()

	cfg := LoadConfig(setup, ssltls)

	tlsCfg := SSLTLSConfig{}
	err := base.ConvertFromStructpb(ssltls, &tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert SSL/TLS config: %w", err)
	}

	DBEndpoint := fmt.Sprintf("%v:%v", cfg.DBHost, cfg.DBPort)
	var db *sqlx.DB
	var dsn string
	engineType := enginesType[cfg.DBEngine]

	var caFilePath, certFilePath, keyFilePath string
	defer func() {
		if caFilePath != "" {
			os.Remove(caFilePath)
		}
		if certFilePath != "" {
			os.Remove(certFilePath)
		}
		if keyFilePath != "" {
			os.Remove(keyFilePath)
		}
	}()

	switch cfg.DBSSLTLSType {
	case "TLS":
		decodedCA, err := base64.StdEncoding.DecodeString(tlsCfg.CA)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CA: %w", err)
		}
		caFilePath, err = createTempFile(decodedCA)
		if err != nil {
			return nil, fmt.Errorf("failed to create CA temp file: %w", err)
		}

		engine := enginesTLS[cfg.DBEngine]
		if cfg.DBEngine == "MySQL" {
			dsn = fmt.Sprintf(engine, cfg.DBUsername, cfg.DBPassword, DBEndpoint, cfg.DBName)
			mysqltlsconfig, err := tlsConfig(caFilePath, "", "")
			if err != nil {
				return nil, fmt.Errorf("failed to create MySQL TLS config: %w", err)
			}
			err = mysql.RegisterTLSConfig("custom", mysqltlsconfig)
			if err != nil {
				return nil, err
			}
		} else {
			dsn = fmt.Sprintf(engine, cfg.DBUsername, cfg.DBPassword, DBEndpoint, cfg.DBName, caFilePath)
		}

	case "mTLS":
		decodedCA, err := base64.StdEncoding.DecodeString(tlsCfg.CA)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CA: %w", err)
		}
		decodedCert, err := base64.StdEncoding.DecodeString(tlsCfg.Cert)
		if err != nil {
			return nil, fmt.Errorf("failed to decode cert: %w", err)
		}
		decodedKey, err := base64.StdEncoding.DecodeString(tlsCfg.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to decode key: %w", err)
		}

		caFilePath, err = createTempFile(decodedCA)
		if err != nil {
			return nil, fmt.Errorf("failed to create CA temp file: %w", err)
		}
		certFilePath, err = createTempFile(decodedCert)
		if err != nil {
			return nil, fmt.Errorf("failed to create cert temp file: %w", err)
		}
		keyFilePath, err = createTempFile(decodedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create key temp file: %w", err)
		}

		engine := enginesMTLS[cfg.DBEngine]
		if cfg.DBEngine == "MySQL" {
			dsn = fmt.Sprintf(engine, cfg.DBUsername, cfg.DBPassword, DBEndpoint, cfg.DBName)
			mysqltlsconfig, err := tlsConfig(caFilePath, certFilePath, keyFilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to create MySQL mTLS config: %w", err)
			}
			err = mysql.RegisterTLSConfig("custom", mysqltlsconfig)
			if err != nil {
				return nil, err
			}
		} else {
			dsn = fmt.Sprintf(engine, cfg.DBUsername, cfg.DBPassword, DBEndpoint, cfg.DBName, caFilePath, certFilePath, keyFilePath)
		}

	case "NO TLS":
		engine := engines[cfg.DBEngine]
		dsn = fmt.Sprintf(engine, cfg.DBUsername, cfg.DBPassword, DBEndpoint, cfg.DBName)

	default:
		return nil, fmt.Errorf("unsupported TLS type: %s", cfg.DBSSLTLSType)
	}

	db, err = sqlx.Connect(engineType, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func getEngine(setup *structpb.Struct) string {
	return setup.GetFields()["engine"].GetStringValue()
}
func getUsername(setup *structpb.Struct) string {
	return setup.GetFields()["username"].GetStringValue()
}
func getPassword(setup *structpb.Struct) string {
	return setup.GetFields()["password"].GetStringValue()
}
func getDatabaseName(setup *structpb.Struct) string {
	return setup.GetFields()["database-name"].GetStringValue()
}
func getHost(setup *structpb.Struct) string {
	return setup.GetFields()["host"].GetStringValue()
}
func getPort(setup *structpb.Struct) string {
	return strconv.Itoa(int(setup.GetFields()["port"].GetNumberValue()))
}
func getSSLTLSType(ssltls *structpb.Struct) string {
	return ssltls.GetFields()["ssl-tls-type"].GetStringValue()
}
