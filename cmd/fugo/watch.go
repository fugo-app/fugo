package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/runcitrus/fugo/internal/agent"
)

type appInstance struct {
	Agents map[string]*agent.Agent
}

func (a *appInstance) loadAgents(configPath string) error {
	agentsDir := filepath.Join(configPath, "agents")

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return err
	}

	a.Agents = make(map[string]*agent.Agent)

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		entryName := entry.Name()
		ext := filepath.Ext(entryName)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		name := strings.TrimSuffix(entryName, ext)

		filePath := filepath.Join(agentsDir, entryName)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read config (%s): %w", filePath, err)
		}

		agentConfig := new(agent.Agent)
		if err := yaml.Unmarshal(data, agentConfig); err != nil {
			return fmt.Errorf("parse config (%s): %w", filePath, err)
		}

		if err := agentConfig.Init(name); err != nil {
			return fmt.Errorf("init agent (%s): %w", name, err)
		}

		a.Agents[name] = agentConfig
	}

	return nil
}

func (a *appInstance) Init(configDir string) error {
	if err := a.loadAgents(configDir); err != nil {
		return fmt.Errorf("loading agents: %w", err)
	}

	return nil
}

func (a *appInstance) Start() {
	// Start all agents
	for _, agent := range a.Agents {
		agent.Start()
	}
}

func (a *appInstance) Stop() {
	for _, agent := range a.Agents {
		agent.Stop()
	}
}

var configFile string

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Start the watch agent",
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(0)

		a := new(appInstance)
		if err := a.Init(configFile); err != nil {
			log.Fatalln("failed to init app:", err)
		}

		a.Start()

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

		<-signalCh

		a.Stop()
	},
}

func init() {
	watchCmd.LocalFlags().StringVar(&configFile, "config", "/etc/fugo", "Path to the configuration directory")
}
