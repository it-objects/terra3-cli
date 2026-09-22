package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ApiUrl           string `mapstructure:"api_url"`
	CognitoDomain    string `mapstructure:"cognito_domain"`
	CognitoClientID  string `mapstructure:"cognito_client_id"`
	IdentityProvider string `mapstructure:"identity_provider"`
	AwsRegion        string `mapstructure:"aws_region"`
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(home, ".terra3")
	_ = os.MkdirAll(configDir, 0755)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
	viper.SetEnvPrefix("TERRA3")
	viper.AutomaticEnv()

	viper.SetDefault("api_url", "https://platform.terra3.io")
	viper.SetDefault("aws_region", "eu-central-1")
	viper.SetDefault("cognito_domain", "https://login.terra3.io")
	viper.SetDefault("cognito_client_id", "6jei2g11mqufr94f1vkn2jor43")
	viper.SetDefault("identity_provider", "EntraID")

	_ = viper.ReadInConfig()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
