package config

import (
	"sync"
)

// Config holds all configuration parameters
type Config struct {
	Server     string
	ProjectID  string
	NodeID     string
	GaiaSecDir string
	DaemonMode bool
	ProxyMode  bool
	Version    bool
	Logflags   string
	Socks5Addr string
	AutoHook   bool
}

var (
	instance *Config
	once     sync.Once
)

// GetInstance returns the singleton instance of Config
func GetInstance() *Config {
	once.Do(func() {
		instance = &Config{
			GaiaSecDir: "/gaiasec",
			Logflags:   "log.LstdFlags",
		}
	})
	return instance
}
