// Package app contains the services exposed to the frontend.
package app

import (
	"runtime"

	"MaaGO/internal/apppath"
	"MaaGO/internal/config"
	MaaVersion "MaaGO/internal/version"
)

// Service is the root Wails service. Frontend bindings are generated from its
// exported methods, so keep the surface small and intentional.
type Service struct{}

// NewService creates the application service.
func NewService() *Service {
	return &Service{}
}

// AppInfo describes the running build.
type AppInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	DataDir   string `json:"dataDir"`
}

// GetAppInfo returns build and environment metadata for the About page.
func (s *Service) GetAppInfo() AppInfo {
	return AppInfo{
		Name:      "MaaGO",
		Version:   MaaVersion.Version,
		Commit:    MaaVersion.Commit,
		BuildDate: MaaVersion.BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		DataDir:   apppath.Root(),
	}
}

// GetConfig returns the persisted application configuration.
func (s *Service) GetConfig() (config.AppConfig, error) {
	return config.Load()
}

// SaveConfig persists the application configuration.
func (s *Service) SaveConfig(cfg config.AppConfig) error {
	return config.Update(func(current *config.AppConfig) {
		*current = cfg
	})
}

// OpenDataDir opens the user data directory in Explorer.
func (s *Service) OpenDataDir() error {
	return openPath(apppath.Root())
}
