package dashboard

import (
	"fmt"
	"log/slog"
	"os"

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
		ListenAddr:        ":8080",
		WKHTMLToImagePath: "wkhtmltoimage",
		DashboardDir:      "dashboards",
	}
}

type Config struct {
	Log               LogConfig `toml:"log"`
	ListenAddr        string    `toml:"listen_addr"`
	WKHTMLToImagePath string    `toml:"wkhtmltoimage_path"`
	DashboardDir      string    `toml:"dashboard_dir"`
}

func (c Config) String() string {
	return fmt.Sprintf("Log: %s\n ListenAddr: %s\n WKHTMLToImagePath: %s\n DashboardDir: %s\n",
		c.Log,
		c.ListenAddr,
		c.WKHTMLToImagePath,
		c.DashboardDir,
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
	return fmt.Sprintf("\n  Level: %s\n  Format: %s\n  AddSource: %t\n  NoColor: %t",
		c.Level,
		c.Format,
		c.AddSource,
		c.NoColor,
	)
}
