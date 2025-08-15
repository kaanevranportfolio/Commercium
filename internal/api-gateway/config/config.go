package config

import (
	"github.com/kaanevranportfolio/Commercium/pkg/config"
)

// Load loads the API Gateway configuration
func Load() (*config.Config, error) {
	return config.Load()
}
