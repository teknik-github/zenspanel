package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Agent       AgentConfig
	Paths       PathsConfig
	LetsEncrypt LetsEncryptConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr string
}

type JWTConfig struct {
	Secret        string
	Expiry        string
	RefreshExpiry string `mapstructure:"refresh_expiry"`
}

type AgentConfig struct {
	Socket      string
	SocketGroup string `mapstructure:"socket_group"`
}

type PathsConfig struct {
	HomeBase    string `mapstructure:"home_base"`
	NginxConf   string `mapstructure:"nginx_conf"`
	SSLBase     string `mapstructure:"ssl_base"`
	BackupBase  string `mapstructure:"backup_base"`
	PHPPoolBase string `mapstructure:"php_pool_base"`
}

type LetsEncryptConfig struct {
	Email   string
	Staging bool
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/zenspanel/")
	viper.AddConfigPath(".")

	viper.SetDefault("server.host", "127.0.0.1")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("redis.addr", "127.0.0.1:6379")
	viper.SetDefault("agent.socket", "/run/zenspanel/agent.sock")
	viper.SetDefault("agent.socket_group", "zenspanel")
	viper.SetDefault("paths.home_base", "/home/zenspanel")
	viper.SetDefault("paths.nginx_conf", "/etc/nginx/zenspanel")
	viper.SetDefault("paths.ssl_base", "/etc/nginx/ssl/zenspanel")
	viper.SetDefault("paths.backup_base", "/var/backups/zenspanel")
	viper.SetDefault("paths.php_pool_base", "/etc/php")
	viper.SetDefault("jwt.expiry", "24h")
	viper.SetDefault("jwt.refresh_expiry", "720h")
	viper.SetDefault("letsencrypt.staging", false)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
