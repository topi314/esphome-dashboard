package dashboard

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

func LoadConfig(cfgPath string) (Config, error) {
	file, err := os.Open(cfgPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	cfg := defaultConfig()
	if _, err = toml.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to decode config file: %w", err)
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Log: LogConfig{
			Level:     slog.LevelInfo,
			Format:    LogFormatText,
			AddSource: false,
			NoColor:   false,
		},
		ListenAddr:   "",
		ListenPort:   8080,
		DashboardDir: "dashboards",
	}
}

type Config struct {
	Dev           bool                 `toml:"dev"`
	ListenAddr    string               `toml:"listen_addr"`
	ListenPort    int                  `toml:"listen_port"`
	DashboardDir  string               `toml:"dashboard_dir"`
	Log           LogConfig            `toml:"log"`
	HomeAssistant *HomeAssistantConfig `toml:"home_assistant"`
}

func (c Config) String() string {
	return fmt.Sprintf("Dev: %t\nListenAddr: %s\nDashboardDir: %s\nLog: %s\nHomeAssistant: %v",
		c.Dev,
		c.ListenAddr,
		c.DashboardDir,
		c.Log,
		c.HomeAssistant,
	)
}

type LogFormat string

const (
	LogFormatJSON   LogFormat = "json"
	LogFormatText   LogFormat = "text"
	LogFormatLogFMT LogFormat = "log-fmt"
)

type LogConfig struct {
	Level     slog.Level `toml:"level"`
	Format    LogFormat  `toml:"format"`
	AddSource bool       `toml:"add_source"`
	NoColor   bool       `toml:"no_color"`
}

func (c LogConfig) String() string {
	return fmt.Sprintf("\n Level: %s\n Format: %s\n AddSource: %t\n NoColor: %t",
		c.Level,
		c.Format,
		c.AddSource,
		c.NoColor,
	)
}

type HomeAssistantConfig struct {
	Host   string `toml:"host"`
	Port   int    `toml:"port"`
	Secure bool   `toml:"secure"`
	Token  string `toml:"token"`
}

func (c HomeAssistantConfig) URL() string {
	scheme := "http"
	if c.Secure {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s:%d", scheme, c.Host, c.Port)
}

func (c HomeAssistantConfig) String() string {
	return fmt.Sprintf("\n Host: %s\n Port: %d\n Secure: %t\n Token: %s",
		c.Host,
		c.Port,
		c.Secure,
		strings.Repeat("*", len(c.Token)),
	)
}
