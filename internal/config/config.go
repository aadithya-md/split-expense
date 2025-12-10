package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServiceName string           `mapstructure:"SERVICE_NAME"`
	HttpServer  HttpServerConfig `mapstructure:"HTTP_SERVER"`
}

type HttpServerConfig struct {
	Address      string        `mapstructure:"ADDRESS"`
	Port         string        `mapstructure:"PORT"`
	ReadTimeout  time.Duration `mapstructure:"READ_TIMEOUT"`
	WriteTimeout time.Duration `mapstructure:"WRITE_TIMEOUT"`
	IdleTimeout  time.Duration `mapstructure:"IDLE_TIMEOUT"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	v.AddConfigPath("./config")
	v.SetConfigName("default")
	v.SetConfigType("yaml")

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
