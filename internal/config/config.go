// Package config provides configuration loading and management.
package config

import (
	"fmt"
	"time"

	projectroot "github.com/yannick2025-tech/gwc-projectroot"
	"github.com/yannick2025-tech/gwc-configor"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Log      LogConfig      `yaml:"log"`
	Database DatabaseConfig `yaml:"database"`
	Protocol ProtocolConfig `yaml:"protocol"`
}

// ServerConfig TCP/HTTP服务配置
type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	HTTPPort        int           `yaml:"http_port"`       // API 接口端口
	WebPort         int           `yaml:"web_port"`         // Web 静态页面端口
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	HeartbeatTimeout time.Duration `yaml:"heartbeat_timeout"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string        `yaml:"driver"`
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ShowSQL         bool          `yaml:"show_sql"`
	LogLevel        string        `yaml:"log_level"`
}

// ProtocolConfig 协议配置
type ProtocolConfig struct {
	Name              string `yaml:"name"`
	Version           string `yaml:"version"`
	FixedKey          string `yaml:"fixed_key"`
	HeartbeatCycle    int    `yaml:"heartbeat_cycle"`     // 心跳上报周期(秒)
	TimeSyncCycle     int    `yaml:"time_sync_cycle"`     // 对时周期(分钟)
	ChargingDataCycle int    `yaml:"charging_data_cycle"` // 充电数据上报周期(秒)
	BMSDataCycle      int    `yaml:"bms_data_cycle"`      // BMS数据上报周期(秒)
}

var globalConfig *Config

// Load 加载配置
func Load() (*Config, error) {
	configPath := projectroot.ResolvePath("configs/config.yaml")

	mgr, err := configor.NewManager(configor.Config{
		ConfigFile: configPath,
		ConfigType: "yaml",
	})
	if err != nil {
		return nil, fmt.Errorf("create config manager failed: %w", err)
	}

	if err := mgr.Read(); err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	cfg := &Config{}
	if err := mgr.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}

	// 设置默认值
	setDefaults(cfg)

	globalConfig = cfg
	return cfg, nil
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

func setDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8888
	}
	if cfg.Server.HTTPPort == 0 {
		cfg.Server.HTTPPort = 9090
	}
	if cfg.Server.WebPort == 0 {
		cfg.Server.WebPort = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}
	if cfg.Server.HeartbeatTimeout == 0 {
		cfg.Server.HeartbeatTimeout = 3 * time.Minute
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 200
	}
	if cfg.Log.MaxBackups == 0 {
		cfg.Log.MaxBackups = 20
	}
	if cfg.Log.MaxAge == 0 {
		cfg.Log.MaxAge = 30
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "mysql"
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 100
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 10
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = time.Hour
	}
	if cfg.Protocol.Name == "" {
		cfg.Protocol.Name = "standard"
	}
	if cfg.Protocol.Version == "" {
		cfg.Protocol.Version = "06"
	}
	if cfg.Protocol.HeartbeatCycle == 0 {
		cfg.Protocol.HeartbeatCycle = 30
	}
	if cfg.Protocol.TimeSyncCycle == 0 {
		cfg.Protocol.TimeSyncCycle = 60
	}
	if cfg.Protocol.ChargingDataCycle == 0 {
		cfg.Protocol.ChargingDataCycle = 30
	}
	if cfg.Protocol.BMSDataCycle == 0 {
		cfg.Protocol.BMSDataCycle = 30
	}
}
