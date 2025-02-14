package dashboard

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/adrg/frontmatter"
)

type DashboardConfig struct {
	Height        int                          `toml:"height"`
	Width         int                          `toml:"width"`
	Base          string                       `toml:"base"`
	Pages         []string                     `toml:"pages"`
	HomeAssistant DashboardHomeAssistantConfig `toml:"home_assistant"`
}

type DashboardHomeAssistantConfig struct {
	Entities  []EntityConfig   `toml:"entities"`
	Calendars []CalendarConfig `toml:"calendars"`
	Services  []ServiceConfig  `toml:"services"`
}

type EntityConfig struct {
	Name string `toml:"name"`
	ID   string `toml:"id"`
}

type CalendarConfig struct {
	Name           string   `toml:"name"`
	IDs            []string `toml:"ids"`
	Days           int      `toml:"days"`
	MaxEvents      int      `toml:"max_events"`
	SkipPastEvents bool     `toml:"skip_past_events"`
}

type ServiceConfig struct {
	Name           string         `toml:"name"`
	Domain         string         `toml:"domain"`
	Service        string         `toml:"service"`
	ReturnResponse bool           `toml:"return_response"`
	Data           map[string]any `toml:"data"`
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
	Name  string
	Index int
	Vars  map[string]any
	Body  []byte
}

type Base struct {
	Vars      map[string]any
	Body      []byte
	PageIndex int
	Pages     []Page
	Config    DashboardConfig
}

func (s *Server) loadDashboard(dashboard string, pageIndex int) (*Base, error) {
	config, err := s.getDashboardConfig(dashboard)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard config: %w", err)
	}

	if pageIndex < 0 || pageIndex >= len(config.Pages) {
		return nil, fmt.Errorf("invalid page index: %d", pageIndex)
	}

	baseFile, err := os.Open(filepath.Join(s.cfg.DashboardDir, dashboard, config.Base))
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer baseFile.Close()

	var baseFrontmatter map[string]any
	baseBody, err := frontmatter.Parse(baseFile, &baseFrontmatter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	var pages []Page
	for i, pageName := range config.Pages {
		page, err := s.loadPage(dashboard, i, pageName)
		if err != nil {
			return nil, fmt.Errorf("failed to load page: %w", err)
		}
		pages = append(pages, *page)
	}

	return &Base{
		Vars:      baseFrontmatter,
		Body:      baseBody,
		PageIndex: pageIndex,
		Pages:     pages,
		Config:    *config,
	}, nil
}

func (s *Server) loadPage(dashboard string, i int, pageName string) (*Page, error) {
	pageFile, err := os.Open(filepath.Join(s.cfg.DashboardDir, dashboard, pageName))
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer pageFile.Close()

	var pageFrontmatter map[string]any
	pageBody, err := frontmatter.Parse(pageFile, &pageFrontmatter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	return &Page{
		Name:  pageName,
		Index: i,
		Vars:  pageFrontmatter,
		Body:  pageBody,
	}, nil
}
