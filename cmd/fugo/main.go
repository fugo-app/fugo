package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/fugo-app/fugo/internal/agent"
	"github.com/fugo-app/fugo/internal/server"
	"github.com/fugo-app/fugo/internal/source/file"
	"github.com/fugo-app/fugo/internal/storage"
)

var Version = "0.0.0"

type appInstance struct {
	Server    server.ServerConfig   `yaml:"server"`
	Storage   storage.StorageConfig `yaml:"storage"`
	FileInput file.FileConfig       `yaml:"file_input"`

	agents map[string]*agent.Agent
}

func main() {
	versionFlag := pflag.Bool("version", false, "Print version")
	helpFlag := pflag.Bool("help", false, "Print this help")
	configFlag := pflag.StringP("config", "c", "/etc/fugo/config.yaml", "Path to config file")
	pflag.Parse()

	if *versionFlag {
		fmt.Println("Fugo version", Version)
		os.Exit(0)
	}

	if *helpFlag {
		fmt.Printf("Fugo is log parsing and processing agent\n\n")
		pflag.Usage()
		os.Exit(0)
	}

	log.SetFlags(0)

	a := new(appInstance)
	if err := a.start(*configFlag); err != nil {
		log.Fatalln("failed to init app:", err)
	}
	defer a.stop()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	<-signalCh
}

func (a *appInstance) loadAgents(configPath string) error {
	agentsDir := filepath.Join(configPath, "agents")

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return err
	}

	a.agents = make(map[string]*agent.Agent)

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

		agent := new(agent.Agent)
		if err := yaml.Unmarshal(data, agent); err != nil {
			return fmt.Errorf("parse config (%s): %w", filePath, err)
		}

		if err := agent.Init(name, &a.Storage); err != nil {
			return fmt.Errorf("init agent (%s): %w", name, err)
		}

		if err := a.Storage.Migrate(name, agent.Fields); err != nil {
			return fmt.Errorf("migrate agent (%s): %w", name, err)
		}

		a.agents[name] = agent
	}

	return nil
}

func (a *appInstance) start(configFile string) error {
	configDir := filepath.Dir(configFile)

	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(configData, a); err != nil {
		return fmt.Errorf("parse config (%s): %w", configFile, err)
	}

	if err := a.Storage.Open(); err != nil {
		return fmt.Errorf("open storage: %w", err)
	}

	if err := a.Server.Open(&a.Storage); err != nil {
		return fmt.Errorf("open server: %w", err)
	}

	if err := a.FileInput.Open(); err != nil {
		return fmt.Errorf("open file-based input: %w", err)
	}

	if err := a.loadAgents(configDir); err != nil {
		return fmt.Errorf("loading agents: %w", err)
	}

	// Start all agents
	for _, agent := range a.agents {
		agent.Start()
	}

	return nil
}

func (a *appInstance) stop() {
	if err := a.Server.Close(); err != nil {
		log.Println("failed to close server:", err)
	}

	for _, agent := range a.agents {
		agent.Stop()
	}

	if err := a.Storage.Close(); err != nil {
		log.Println("failed to close storage:", err)
	}

	if err := a.FileInput.Close(); err != nil {
		log.Println("failed to close file-based input:", err)
	}
}
