package config

import (
	"sync"
)

// Config holds all configuration parameters
type Config struct {
	ServerURL  string
	ProjectID  string
	NodeID     string
	SothothDir string
	DaemonMode bool
	Proxy      bool
	Version    bool
	Logflags   string
}

var (
	instance *Config
	once     sync.Once
)

// GetInstance returns the singleton instance of Config
func GetInstance() *Config {
	once.Do(func() {
		instance = &Config{
			SothothDir: "/sothoth",
			Logflags:   "log.LstdFlags",
		}
	})
	return instance
}
