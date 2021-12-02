package config

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"time"

	"github.com/GGP1/groove/internal/log"
	"github.com/bradfitz/gomemcache/memcache"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Config represents groove's configuration.
type Config struct {
	Admins        map[string]interface{}
	TLS           TLS
	Logger        Logger
	Notifications Notifications
	Dgraph        Dgraph
	Memcached     Memcached
	Postgres      Postgres
	Redis         Redis
	Server        Server
	Sessions      Sessions
	RateLimiter   RateLimiter
	Development   bool
}

// Dgraph configuration.
type Dgraph struct {
	Host            string
	TLSCertificates []tls.Certificate
	Port            int
}

// Logger contains Zap's configurations.
type Logger struct {
	OutFiles []string
}

// Memcached configuration.
type Memcached struct {
	Servers         []string
	MaxIdleConns    int
	Timeout         time.Duration
	ItemsExpiration int32
}

// Notifications configuration.
type Notifications struct {
	CredentialsFile string
	MaxRetries      int
}

// Postgres configuration.
type Postgres struct {
	Username        string
	Password        string
	Host            string
	Port            string
	Name            string
	SSLMode         string
	SSLRootCert     string
	SSLCert         string
	SSLKey          string
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	MetricsRate     time.Duration
}

// RateLimiter configuration.
type RateLimiter struct {
	Rate int
}

// Redis configuration.
type Redis struct {
	Host            string
	Port            string
	Password        string
	TLSCertificates []tls.Certificate
	PoolSize        int
	MinIdleConns    int
	MetricsRate     time.Duration
}

// Server configuration.
type Server struct {
	Host            string
	Port            string
	TLSCertificates []tls.Certificate
	LetsEncrypt     struct {
		Cache     string
		Hosts     []string
		Enabled   bool
		AcceptTOS bool
	}
	Timeout struct {
		Read     time.Duration
		Write    time.Duration
		Shutdown time.Duration
	}
}

// Sessions configuration.
type Sessions struct {
	VerifyEmails bool
	Expiration   time.Duration
}

// TLS certificate and keyfile.
type TLS struct {
	Certfile string
	Keyfile  string
}

// New creates a new configuration.
func New() (Config, error) {
	configUsed := "default"
	configPath := os.Getenv("GROOVE_CONFIG")
	if configPath != "" {
		ext := filepath.Ext(configPath)
		if ext == "" || ext == "." {
			return Config{}, errors.New("\"GROOVE_CONFIG\" environment variable must have an extension")
		}
		viper.SetConfigType(ext[1:])
		configUsed = "customized"
	} else {
		var err error
		configPath, err = defaultConfig()
		if err != nil {
			return Config{}, err
		}
	}

	viper.AutomaticEnv()
	for k, v := range envVars {
		viper.BindEnv(k, v)
	}

	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return Config{}, errors.Wrap(err, "writing configuration to memory")
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return Config{}, errors.Wrap(err, "unmarshal configuration")
	}

	if err := validateValues(config); err != nil {
		return Config{}, err
	}

	if err := log.Setup(config.Development, config.Logger.OutFiles); err != nil {
		return Config{}, err
	}

	var certificates []tls.Certificate
	if config.TLS.Certfile != "" && config.TLS.Keyfile != "" {
		cert, err := tls.LoadX509KeyPair(config.TLS.Certfile, config.TLS.Keyfile)
		if err != nil {
			return Config{}, errors.Wrap(err, "loading x509 key pair")
		}
		certificates = []tls.Certificate{cert}
	}

	config.Dgraph.TLSCertificates = certificates
	config.Redis.TLSCertificates = certificates
	config.Server.TLSCertificates = certificates

	log.Sugar().Infof("Using %s configuration: %s", configUsed, viper.ConfigFileUsed())
	return *config, nil
}

func defaultConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "finding home directory")
	}
	home = filepath.Join(home, ".groove")

	if err := os.MkdirAll(home, 0700); err != nil {
		return "", errors.Wrap(err, "creating the directory")
	}

	configPath := filepath.Join(home, "groove.yml")
	if _, err := os.Stat(configPath); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		for k, v := range defaults {
			viper.SetDefault(k, v)
		}

		if err := viper.WriteConfigAs(configPath); err != nil {
			return "", errors.Wrap(err, "writing configuration file")
		}
	}

	viper.SetConfigType("yaml")
	return configPath, nil
}

func validateValues(config *Config) error {
	if !config.Development && config.RateLimiter.Rate == 0 {
		return errors.New("no rate limit provided in production mode")
	}

	if config.Redis.MetricsRate != 0 && config.Redis.MetricsRate < 1 {
		return errors.New("redis metrics rate too low")
	}
	if config.Postgres.MetricsRate != 0 && config.Postgres.MetricsRate < 1 {
		return errors.New("postgres metrics rate too low")
	}

	return nil
}

var (
	defaults = map[string]interface{}{
		"admins":      map[string]interface{}{},
		"development": true,
		"dgraph": map[string]interface{}{
			"host": "localhost",
			"port": 9080,
		},
		"logger": map[string]interface{}{
			"outFiles": []string{},
		},
		"memcached": map[string]interface{}{
			"itemsExpiration": 0,
			"maxIdleConns":    memcache.DefaultMaxIdleConns,
			"servers":         []string{"localhost:11211"},
			"timeout":         memcache.DefaultTimeout,
		},
		"notifications": map[string]interface{}{
			"credentialsFile": "/firebase_credentials.json",
			"maxRetries":      5,
		},
		"postgres": map[string]interface{}{
			"host":            "postgres",
			"port":            "5432",
			"name":            "postgres",
			"username":        "postgres",
			"password":        "postgres",
			"sslMode":         "disabled",
			"sslRootCert":     "",
			"sslCert":         "",
			"sslKey":          "",
			"maxIdleConns":    50,
			"maxConnIdleTime": 300, // Seconds
			"metricsRate":     60,  // Seconds
		},
		"rateLimiter": map[string]interface{}{
			"rate": 5,
		},
		"redis": map[string]interface{}{
			"host":         "localhost",
			"port":         6379,
			"password":     "redis",
			"poolSize":     0, // Use default (GOMAXPROCS * 10)
			"minIdleConns": 10,
			"metricsRate":  60, // Seconds
		},
		"secrets": map[string]interface{}{
			"encryption": "{X]_?4-6)hgp(P_w9nTO8f =2Gki",
			"apiKeys":    "u3xLK_7HBf!s@1p-*Gj/]oUgQ>E4",
		},
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 4000,
			"letsEncrypt": map[string]interface{}{
				"enabled":   false,
				"acceptTOS": false,
				"cache":     "",
				"hosts":     []string{},
			},
			"timeout": map[string]interface{}{
				"read":     5, // Seconds
				"write":    5, // Seconds
				"shutdown": 5, // Seconds
			},
		},
		"sessions": map[string]interface{}{
			"verifyEmails": false,
			"expiration":   "168h", // 7 days
		},
		"tls": map[string]interface{}{
			"certFile": "",
			"keyFile":  "",
		},
	}

	envVars = map[string]string{
		// Admins
		"admins": "ADMINS",
		// Development
		"development": "DEVELOPMENT",
		// Dgraph
		"dgraph.host": "DGRAPH_HOST",
		"dgraph.port": "DGRAPH_PORT",
		// Logger
		"logger.outFiles": "LOGGER_OUT_FILES",
		// Memcached
		"memcached.itemsExpiration": "MEMCACHED_ITEMS_EXPIRATION",
		"memcached.maxIdleConns":    "MEMCACHED_MAX_IDLE_CONS",
		"memcached.servers":         "MEMCACHED_SERVERS",
		"memcached.timeout":         "MEMCACHED_TIMEOUT",
		// Notifications
		"notifications.credentialsFile": "NOTIFICATIONS_CREDENTIALS_FILE",
		"notifications.maxRetries":      "NOTIFICATIONS_MAX_RETRIES",
		// Postgres
		"postgres.username":        "POSTGRES_USERNAME",
		"postgres.password":        "POSTGRES_PASSWORD",
		"postgres.host":            "POSTGRES_HOST",
		"postgres.port":            "POSTGRES_PORT",
		"postgres.name":            "POSTGRES_DB",
		"postgres.sslMode":         "POSTGRES_SSL_MODE",
		"postgres.sslRootCert":     "POSTGRES_SSL_ROOT_CERT",
		"postgres.sslCert":         "POSTGRES_SSL_CERT",
		"postgres.sslKey":          "POSTGRES_SSL_KEY",
		"postgres.maxIdleConns":    "POSTGRES_MAX_IDLE_CONNS",
		"postgres.maxConnIdleTime": "POSTGRES_MAX_CONN_IDLE_TIME",
		"postgres.metricsRate":     "POSTGRES_METRICS_RATE",
		// Rate limiter
		"rateLimiter.rate": "RATE_LIMITER_RATE",
		// Redis
		"redis.host":         "REDIS_HOST",
		"redis.port":         "REDIS_PORT",
		"redis.password":     "REDIS_PASSWORD",
		"redis.poolSize":     "REDIS_POOL_SIZE",
		"redis.minIdleConns": "REDIS_MIN_IDLE_CONS",
		"redis.metricsRate":  "REDIS_METRICS_RATE",
		// Secrets
		"secrets.encryption": "SECRETS_ENCRYPTION",
		"secrets.apiKeys":    "SECRETS_API_KEYS",
		// Server
		"server.host":                  "SV_HOST",
		"server.port":                  "SV_PORT",
		"server.letsEncrypt.enabled":   "SV_LETSENCRYPT_ENABLED",
		"server.letsEncrypt.acceptTOS": "SV_LETSENCRYPT_ACCEPT_TOS",
		"server.letsEncrypt.cache":     "SV_LETSENCRYPT_CACHE",
		"server.letsEncrypt.hosts":     "SV_LETSENCRYPT_HOSTS",
		"server.timeout.read":          "SV_TIMEOUT_READ",
		"server.timeout.write":         "SV_TIMEOUT_WRITE",
		"server.timeout.shutdown":      "SV_TIMEOUT_SHUTDOWN",
		// Sessions
		"sessions.verifyEmails": "SESSIONS_VERIFY_EMAILS",
		"sessions.expiration":   "SESSIONS_EXPIRATION",
		// TLS
		"tls.certFile": "TLS_CERT_FILE",
		"tls.keyFile":  "TLS_KEY_FILE",
	}
)
