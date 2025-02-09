package dashboard

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/adrg/frontmatter"
)

type DashboardConfig struct {
	Name      string   `toml:"name"`
	Height    int      `toml:"height"`
	Width     int      `toml:"width"`
	Quality   int      `toml:"quality"`
	AssetsDir string   `toml:"assets_dir"`
	Pages     []string `toml:"pages"`
}

func (s *Server) getDashboardConfig(dashboard string) (*DashboardConfig, error) {
	configFile, err := os.Open(filepath.Join(s.cfg.DashboardDir, dashboard, "config.toml"))
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer configFile.Close()

	var config DashboardConfig
	if _, err = toml.NewDecoder(configFile).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

func (s *Server) getNextPageIndex(dashboard string, lastPage int, action Action) (int, error) {
	config, err := s.getDashboardConfig(dashboard)
	if err != nil {
		return 0, fmt.Errorf("failed to get dashboard config: %w", err)
	}

	var pageIndex int
	switch action {
	case ActionRefresh:
		pageIndex = lastPage
	case ActionNextPage:
		if lastPage+1 >= len(config.Pages) {
			pageIndex = 0
		} else {
			pageIndex = lastPage + 1
		}
	case ActionLastPage:
		pageIndex = len(config.Pages) - 1
	case ActionPrevPage:
		if lastPage-1 < 0 {
			pageIndex = len(config.Pages) - 1
		} else {
			pageIndex = lastPage - 1
		}
	case ActionFirstPage:
		pageIndex = 0
	default:
		return 0, fmt.Errorf("unknown action: %s", action)
	}

	return pageIndex, nil
}

type Page struct {
	Index           int
	Vars            map[string]any
	Body            []byte
	DashboardConfig DashboardConfig
}

func (s *Server) loadPage(dashboard string, pageIndex int) (*Page, error) {
	config, err := s.getDashboardConfig(dashboard)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard config: %w", err)
	}

	if pageIndex < 0 || pageIndex >= len(config.Pages) {
		return nil, fmt.Errorf("invalid page index: %d", pageIndex)
	}

	pageFile, err := os.Open(filepath.Join(s.cfg.DashboardDir, dashboard, config.Pages[pageIndex]))
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer pageFile.Close()

	var pf map[string]any
	body, err := frontmatter.Parse(pageFile, &pf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	return &Page{
		Index:           pageIndex,
		Vars:            pf,
		Body:            body,
		DashboardConfig: *config,
	}, nil
}

type RenderOptions struct {
	BinaryPath string
	AssetsDir  string
	Width      int
	Height     int
	Quality    int
}

type RenderData struct {
	Vars map[string]any
}

func (s *Server) renderPage(ctx context.Context, page Page, options RenderOptions) (io.Reader, int, error) {
	t, err := template.New("page").Parse(string(page.Body))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err = t.Execute(&buf, RenderData{
		Vars: page.Vars,
	}); err != nil {
		return nil, 0, fmt.Errorf("failed to execute template: %w", err)
	}

	cmd := exec.CommandContext(ctx, options.BinaryPath,
		"--width", strconv.Itoa(options.Width),
		"--height", strconv.Itoa(options.Height),
		"--quality", strconv.Itoa(options.Quality),
		"--allow", options.AssetsDir,
		"--disable-smart-width",
		"--disable-javascript",
		"--disable-plugins",
		"-f", "png",
		"-", "-",
	)
	cmd.Stdin = &buf
	var imageBuf bytes.Buffer
	cmd.Stdout = &imageBuf
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return nil, 0, fmt.Errorf("failed to run command: %q: %w", stderr.String(), err)
	}

	return s.reencodePNG(&imageBuf)
}

func (s *Server) reencodePNG(r io.Reader) (io.Reader, int, error) {
	decoded, err := png.Decode(r)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode png: %w", err)
	}

	encodedBuf := new(bytes.Buffer)
	if err = s.Encoder.Encode(encodedBuf, decoded); err != nil {
		return nil, 0, fmt.Errorf("failed to encode png: %w", err)
	}

	return encodedBuf, encodedBuf.Len(), nil
}
