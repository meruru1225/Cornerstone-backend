package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

// Cfg 全局可访问的配置实例
var Cfg *Config

// LoadConfig 从文件加载配置并填充到 Cfg
func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("config file not found: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	Cfg = &cfg

	return nil
}
