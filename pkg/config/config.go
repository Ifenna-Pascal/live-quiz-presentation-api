package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds every setting. Values come from app.env, and any environment
// variable with the same name overrides the file (use that in production).
type Config struct {
	Environment   string `mapstructure:"ENVIRONMENT"`
	ServerAddress string `mapstructure:"SERVER_ADDRESS"`
	DBDriver      string `mapstructure:"DB_DRIVER"`
	DBSource      string `mapstructure:"DB_SOURCE"`
	MigrationURL  string `mapstructure:"MIGRATION_URL"`

	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`

	// CORSAllowedOrigins is a comma-separated list of websites allowed to
	// call the API, e.g. "https://app.example.com,http://localhost:5173".
	// Empty allows every website.
	CORSAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`
}

const devKeyMarker = "dev-only"

// AllowedOrigins returns CORSAllowedOrigins as a list, without blanks.
func (c Config) AllowedOrigins() []string {
	var origins []string
	for _, origin := range strings.Split(c.CORSAllowedOrigins, ",") {
		// Browsers send origins without a trailing slash.
		if origin = strings.TrimRight(strings.TrimSpace(origin), "/"); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

// IsProduction reports whether ENVIRONMENT is "production".
func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}

// Validate refuses settings that are unsafe or missing.
func (c Config) Validate() error {
	if c.AccessTokenDuration <= 0 {
		return errors.New("ACCESS_TOKEN_DURATION must be set, e.g. 15m")
	}
	if c.RefreshTokenDuration <= c.AccessTokenDuration {
		return errors.New("REFRESH_TOKEN_DURATION must be longer than ACCESS_TOKEN_DURATION, e.g. 168h")
	}
	if c.IsProduction() {
		if strings.Contains(c.TokenSymmetricKey, devKeyMarker) {
			return errors.New("TOKEN_SYMMETRIC_KEY is the development key; set a secret one in the environment")
		}
		if len(c.AllowedOrigins()) == 0 {
			return errors.New("CORS_ALLOWED_ORIGINS must list the frontend's URL in production")
		}
	}
	return nil
}

// LoadConfig reads configuration from path/app.env (optional) and the
// environment.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Register every key so environment variables work without app.env.
	t := reflect.TypeOf(config)
	for i := 0; i < t.NumField(); i++ {
		if key := t.Field(i).Tag.Get("mapstructure"); key != "" {
			_ = viper.BindEnv(key)
		}
	}

	if err = viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return config, fmt.Errorf("read app.env: %w", err)
		}
	}
	if err = viper.Unmarshal(&config); err != nil {
		return config, err
	}
	return config, config.Validate()
}
