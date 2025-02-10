package region

import (
	"github.com/jwalitptl/admin-api/internal/model"
)

type DefaultConfig struct {
	model.RegionConfig
	// Additional fields specific to DefaultConfig
}

func NewDefaultConfig() *DefaultConfig {
	return &DefaultConfig{
		// Initialize with default values
	}
}
