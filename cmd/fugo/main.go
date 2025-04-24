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
	"github.com/fugo-app/fugo/internal/field"
	"github.com/fugo-app/fugo/internal/input/file"
	"github.com/fugo-app/fugo/internal/server"
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
		fmt.Println("Fugo", Version)
		os.Exit(0)
	}

	if *helpFlag {
		fmt.Printf("Fugo is a lightweight log collection and querying agent\n\n")
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

func (a *appInstance) loadAgents(configDir string) error {
	agentsDir := filepath.Join(configDir, "agents")

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

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

		if err := agent.Init(name, a); err != nil {
			return fmt.Errorf("init agent (%s): %w", name, err)
		}

		a.agents[name] = agent
	}

	return nil
}

func (a *appInstance) saveAgents(configDir string) error {
	agentsDir := filepath.Join(configDir, "agents")

	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("create agents directory: %w", err)
	}

	for name, agent := range a.agents {
		agentFile := filepath.Join(agentsDir, name+".yaml")
		agentData, err := yaml.Marshal(agent)
		if err != nil {
			return fmt.Errorf("marshal agent (%s): %w", name, err)
		}
		if err := os.WriteFile(agentFile, agentData, 0644); err != nil {
			return fmt.Errorf("write agent (%s): %w", name, err)
		}
	}

	return nil
}

func (a *appInstance) saveConfig(configFile string) error {
	configDir := filepath.Dir(configFile)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	configData, err := yaml.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	if err := a.saveAgents(configDir); err != nil {
		return fmt.Errorf("save agents: %w", err)
	}

	return nil
}

func (a *appInstance) start(configFile string) error {
	configDir := filepath.Dir(configFile)

	configData, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			libDir := "/var/lib/fugo"
			a.Server.InitDefault()
			a.Storage.InitDefault(libDir)
			a.FileInput.InitDefault(libDir)

			if err := a.saveConfig(configFile); err != nil {
				return fmt.Errorf("save default config: %w", err)
			}
		} else {
			return fmt.Errorf("read config: %w", err)
		}
	}

	if err := yaml.Unmarshal(configData, a); err != nil {
		return fmt.Errorf("parse config (%s): %w", configFile, err)
	}

	if err := a.Storage.Open(); err != nil {
		return fmt.Errorf("open storage: %w", err)
	}

	if err := a.Server.Open(a); err != nil {
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

func (a *appInstance) GetStorage() storage.StorageDriver {
	return a.Storage.GetDriver()
}

func (a *appInstance) GetFields(name string) []*field.Field {
	if agent, ok := a.agents[name]; ok {
		return agent.GetFields()
	}

	return nil
}

func (a *appInstance) GetAgents() []string {
	agentNames := make([]string, 0, len(a.agents))
	for name := range a.agents {
		agentNames = append(agentNames, name)
	}
	return agentNames
}
