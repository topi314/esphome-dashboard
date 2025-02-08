package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/adrg/frontmatter"
)

type DashboardConfig struct {
	Name  string   `toml:"name"`
	Pages []string `toml:"pages"`
}

type Page struct {
	Title string
	Index int
	Count int
	Vars  map[string]any
	Body  []byte
}

type PageFrontmatter struct {
	Title string         `yaml:"title"`
	Vars  map[string]any `yaml:"vars"`
}

func (s *server) getDashboardConfig(dashboard string) (*DashboardConfig, error) {
	configFile, err := os.Open(filepath.Join(s.BaseDir, dashboard, "config.toml"))
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}

	var config DashboardConfig
	if _, err = toml.DecodeReader(configFile, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

func (s *server) getNextPageIndex(dashboard string, lastPage int, action Action) (int, error) {
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

func (s *server) loadPage(dashboard string, pageIndex int) (*Page, error) {
	config, err := s.getDashboardConfig(dashboard)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard config: %w", err)
	}

	pageFile, err := os.Open(filepath.Join(s.BaseDir, dashboard, config.Pages[pageIndex]))
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	var pf PageFrontmatter
	body, err := frontmatter.Parse(pageFile, &pf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	return &Page{
		Title: pf.Title,
		Index: pageIndex,
		Count: len(config.Pages),
		Vars:  pf.Vars,
		Body:  body,
	}, nil
}

type RenderOptions struct {
	BinaryPath string
	AssetsPath string
	Width      int
	Height     int
	Quality    int
}

type ExecuteData struct {
	Title string
	Vars  map[string]any
}

func (s *server) renderPage(ctx context.Context, page Page, options RenderOptions) (io.Reader, error) {
	t, err := template.New("page").Parse(string(page.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err = t.Execute(&buf, ExecuteData{
		Title: page.Title,
		Vars:  page.Vars,
	}); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	cmd := exec.CommandContext(ctx, options.BinaryPath,
		"--width", strconv.Itoa(options.Width),
		"--height", strconv.Itoa(options.Height),
		"--quality", strconv.Itoa(options.Quality),
		"--allow", options.AssetsPath,
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
		return nil, fmt.Errorf("failed to run command: %q: %w", stderr.String(), err)
	}

	return &imageBuf, nil
}
