package config

import (
	"github.com/runcitrus/fugo/internal/agent"
)

type Config struct {
	Agents []agent.Agent `yaml:"agents"`
}
