package config

import (
	"log"

	"github.com/spf13/viper"
)

type CameraConfig struct {
	Name                string  `mapstructure:"name"`
	SnapshotURL         string  `mapstructure:"snapshot_url"`
	ThresholdPercent    float64 `mapstructure:"threshold_percent"`
	MinThresholdPercent float64 `mapstructure:"min_threshold_percent"`
	Cooldown            int64   `mapstructure:"cooldown"`
	Enabled             bool    `mapstructure:"enabled"`
}

type Config struct {
	TelegramToken  string         `mapstructure:"telegram_token"`
	TelegramChatID int64          `mapstructure:"telegram_chat_id"`
	Cameras        []CameraConfig `mapstructure:"cameras"`
}

func LoadFromYAML(path string) Config {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("error reading config file: %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("error unmarshaling config: %v", err)
	}

	return cfg
}
