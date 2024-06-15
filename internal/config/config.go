package config

import (
	"errors"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

var (
	ErrConfigNotLoaded = errors.New("config not loaded")
)

type Environment string

const (
	Production  Environment = "prod"
	Development Environment = "dev"
)

func (e *Environment) SetValue(s string) error {
	*e = Environment(s)
	if *e != Production && *e != Development {
		return configNotLoadedErr(`only "prod" and "dev" environments are allowed`)
	}
	return nil
}

type Config struct {
	App struct {
		Env Environment `yaml:"env" env:"ENV" env-required:""`
	} `yaml:"app" env-prefix:"APP_" env-required:""`

	Server struct {
		Host string `yaml:"host" env:"HOST" env-default:"localhost"`
		Port int    `yaml:"port" env:"PORT" env-default:"8080"`
	} `yaml:"server" env-prefix:"SERVER_"`

	DB struct {
		DSN string `yaml:"dsn" env:"DB_DSN" env-required:""`
	} `yaml:"db" env-prefix:"DB_" env-required:""`

	JWT struct {
		AccessTokenTTL  time.Duration `yaml:"access_token_ttl" env:"ACCESS_TOKEN_TTL" env-default:"2h"`
		RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" env:"REFRESH_TOKEN_TTL" env-default:"24h"`
		Secret          string        `yaml:"secret" env:"SECRET" env-required:""`
	} `yaml:"jwt" env-prefix:"JWT_" env-required:""`
}

func Load(filePath string) (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(filePath, cfg); err != nil {
		return nil, configNotLoadedErr("config not loaded: %w", err)
	}

	return cfg, nil
}

func MustLoad(filePath string) *Config {
	cfg, err := Load(filePath)
	if err != nil {
		panic(err)
	}
	return cfg
}

func configNotLoadedErr(format string, args ...any) error {
	return errors.Join(fmt.Errorf(format, args...), ErrConfigNotLoaded)
}
