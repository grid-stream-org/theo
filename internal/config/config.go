package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/grid-stream-org/go-commons/pkg/bqclient"
	"github.com/grid-stream-org/go-commons/pkg/logger"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/pkg/errors"
)

type Config struct {
	Theo     *Theo            `koanf:"theo"`
	Database *bqclient.Config `koanf:"database"`
	Log      *logger.Config   `koanf:"log"`
}

type Theo struct {
	Timeout      time.Duration `koanf:"timeout"`
	PollInterval time.Duration `koanf:"poll_interval"`
}

func Load() (*Config, error) {
	k := koanf.New(".")

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = filepath.Join("configs", "config.json")
		logger.Default().Info("CONFIG_PATH not set, using default", "path", path)
	}
	if err := k.Load(file.Provider(path), json.Parser()); err != nil {
		return nil, errors.WithStack(err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, errors.WithStack(err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, errors.WithStack(err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	return nil
}
