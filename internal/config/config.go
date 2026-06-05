package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"dario.cat/mergo"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"gopkg.in/yaml.v3"
)

type DiscordConfig struct {
	Token string `yaml:"token"`
}

type LavalinkConfig struct {
	Nodes []disgolink.NodeConfig `yaml:"nodes"`
}

type OllamaSystemPromptConfig struct {
	Name         string `yaml:"name"`         // ¯\_(ツ)_/¯
	Model        string `yaml:"model"`        // override model to use in ollama request. requires model present in ollama
	Exclusive    bool   `yaml:"exclusive"`    // removes system-prompts earlier in the chain Default < Server < Channel
	SystemPrompt string `yaml:"systemPrompt"` // system-prompt to provide when used
}

type OllamaConfig struct {
	Server         string                              `yaml:"server,omitempty"`
	ChatPath       string                              `yaml:"chatPath,omitempty"`
	GeneratePath   string                              `yaml:"generatePath,omitempty"`
	DefaultPrompt  OllamaSystemPromptConfig            `yaml:"defaultPrompt,omitempty"`
	ServerPrompts  map[uint64]OllamaSystemPromptConfig `yaml:"serverPrompts,omitempty"`  // server/channel id as key
	ChannelPrompts map[uint64]OllamaSystemPromptConfig `yaml:"channelPrompts,omitempty"` // server/channel id as key
}

type RobotoConfig struct {
	Discord  *DiscordConfig  `yaml:"discord"`
	Lavalink *LavalinkConfig `yaml:"lavalink"` // Optional
	Ollama   *OllamaConfig   `yaml:"ollama"`
}

func resolve(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path argument can't be empty")
	}

	var err error
	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
	}

	return path, err
}

func load(paths ...string) (*RobotoConfig, error) {
	var errs error

	for i := range paths {
		path, err := resolve(paths[i])
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		file, err := os.ReadFile(path)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		cfg := &RobotoConfig{}
		err = yaml.Unmarshal(file, cfg)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("unmarshal %s: %w", path, err))
			break
		}

		return cfg, nil
	}

	return nil, errs
}

func validate(cfg *RobotoConfig) error {
	var errs error

	if cfg.Discord != nil {
		if cfg.Discord.Token == "" {
			errs = errors.Join(errs, fmt.Errorf("discord config is missing a required token"))
		}
	} else {
		errs = errors.Join(errs, fmt.Errorf("discord config is missing"))
	}

	if cfg.Lavalink != nil {
		nodes := cfg.Lavalink.Nodes
		if len(nodes) == 0 {
			errs = errors.Join(errs, fmt.Errorf("lavalink config must contain a list of nodes"))
		}

		for i := range nodes {
			node := nodes[i]
			if node.Address == "" {
				errs = errors.Join(errs, fmt.Errorf("lavalink config is missing a required address for node %d", i+1))
			}
			if node.Password == "" {
				errs = errors.Join(errs, fmt.Errorf("lavalink config is missing a required password for node %d", i+1))
			}
		}

	}

	return errs
}

func New() (*RobotoConfig, error) {
	// Read config
	cfg, err := load(os.Getenv("BOT_CONFIG_PATH"), "./config.yaml", "./config.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	cfgSecrets, _ := load(os.Getenv("BOT_CONFIG_SECRETS_PATH"), "./config_secrets.yaml", "./config_secrets.yml")

	// Don't attempt merge if we haven't resolvet any secret config
	if cfgSecrets != nil {
		err = mergo.Merge(cfg, cfgSecrets)
		if err != nil {
			return nil, fmt.Errorf("failed to merge config: %w", err)
		}
	}

	err = validate(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to validate merged config: %w", err)
	}

	return cfg, nil
}
