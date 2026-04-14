package main

import (
	"fmt"
	"os"

	"github.com/yannick2025-tech/nts-gater/internal/config"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard"
	logging "github.com/yannick2025-tech/gwc-logging"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger := initLogger(cfg.Log)
	defer logger.Close()

	logger.Info("nts-gater starting...")
	logger.Infof("config loaded: server=%s:%d", cfg.Server.Host, cfg.Server.Port)

	// 初始化协议
	proto := standard.New()
	logger.Infof("protocol: name=%s, version=0x%02X, registered_messages=%d",
		proto.Name(), proto.Version(), len(proto.Registry().AllSpecs()))
}

func initLogger(cfg config.LogConfig) logging.Logger {
	logCfg := &logging.Config{
		LevelStr:      cfg.Level,
		Format:        logging.TextFormat,
		Target:        logging.StdoutTarget,
		Filename:      cfg.Filename,
		MaxSize:       cfg.MaxSize,
		MaxBackups:    cfg.MaxBackups,
		MaxAge:        cfg.MaxAge,
		Compress:      cfg.Compress,
		LocalTime:     true,
		FlushInterval: 5,
	}

	logger, err := logging.NewZapLogger(logCfg)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
	return logger
}
