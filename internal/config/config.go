package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mroctopus/bottie-bot/internal/util"
	"gopkg.in/yaml.v3"
)

type DiscordConfig struct {
	Token string `yaml:"token"`
}

type NodeConfig struct {
	Name     string `yaml:"name"`
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
	Secure   bool   `yaml:"secure"`
}

type LavalinkConfig struct {
	Nodes []NodeConfig `yaml:"nodes"`
}

type RobotoConfig struct {
	Discord  DiscordConfig   `yaml:"discord"`
	Lavalink *LavalinkConfig `yaml:"lavalink"` // Optional
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
	var allErr error

	for i := range paths {
		path, err := resolve(paths[i])
		if err != nil {
			allErr = errors.Join(allErr, err)
			continue
		}

		file, err := os.ReadFile(path)
		if err != nil {
			allErr = errors.Join(allErr, err)
			continue
		}

		cfg := &RobotoConfig{}
		err = yaml.Unmarshal(file, cfg)
		if err != nil {
			allErr = errors.Join(allErr, fmt.Errorf("unmarshal %s: %w", path, err))
			break
		}

		return cfg, nil
	}

	return nil, allErr
}

func merge(src *RobotoConfig, tgt *RobotoConfig) *RobotoConfig {
	if src != nil {
		if src.Discord.Token != "" {
			tgt.Discord.Token = src.Discord.Token
		}

		// NOTE:
		// This looks uuuuuugglys
		if src.Lavalink != nil {
			if tgt.Lavalink == nil {
				tgt.Lavalink = src.Lavalink
			} else {
				if src.Lavalink.Nodes != nil {
					if tgt.Lavalink.Nodes == nil {
						tgt.Lavalink.Nodes = src.Lavalink.Nodes
					} else {
						for i := range src.Lavalink.Nodes {
							if src.Lavalink.Nodes[i].Name != "" {
								tgt.Lavalink.Nodes[i].Name = src.Lavalink.Nodes[i].Name
							}
							if src.Lavalink.Nodes[i].Address != "" {
								tgt.Lavalink.Nodes[i].Address = src.Lavalink.Nodes[i].Address
							}
							if src.Lavalink.Nodes[i].Password != "" {
								tgt.Lavalink.Nodes[i].Password = src.Lavalink.Nodes[i].Password
							}

							tgt.Lavalink.Nodes[i].Secure = src.Lavalink.Nodes[i].Secure
						}

						srcLen := len(src.Lavalink.Nodes)
						tgtLen := len(tgt.Lavalink.Nodes)
						if (srcLen - tgtLen) > 0 {
							for i := tgtLen; i < srcLen; i += 1 {
								tgt.Lavalink.Nodes = append(tgt.Lavalink.Nodes, src.Lavalink.Nodes[i])
							}
						}
					}
				}
			}

		}
	}

	return tgt
}

func validate(cfg *RobotoConfig) error {
	var allErr error

	if cfg.Discord.Token == "" {
		allErr = errors.Join(allErr, fmt.Errorf("discord config is missing a required token"))
	}

	if cfg.Lavalink != nil {
		nodes := cfg.Lavalink.Nodes
		if len(nodes) == 0 {
			allErr = errors.Join(allErr, fmt.Errorf("lavalink config must contain a list of nodes"))
		}

		for i := range nodes {
			node := nodes[i]
			if node.Address == "" {
				allErr = errors.Join(allErr, fmt.Errorf("lavalink config is missing a required address for node %d", i+1))
			}
			if node.Password == "" {
				allErr = errors.Join(allErr, fmt.Errorf("lavalink config is missing a required password for node %d", i+1))
			}
		}

	}

	return allErr
}

func Load() (*RobotoConfig, error) {
	if util.IsDev() {
		_, file, _, _ := runtime.Caller(0)
		root := filepath.Join(filepath.Dir(file), "../../")

		os.Setenv("BOT_CONFIG_PATH", filepath.Join(root, "config.yaml"))
		os.Setenv("BOT_CONFIG_SECRETS_PATH", filepath.Join(root, "config_secrets.yaml"))
	}

	// Read config
	cfg, err := load(os.Getenv("BOT_CONFIG_PATH"), "./config.yaml", "./config.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	cfgSecrets, _ := load(os.Getenv("BOT_CONFIG_SECRETS_PATH"), "./config_secrets.yaml", "./config_secrets.yml")

	err = validate(merge(cfgSecrets, cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to validate merged config: %w", err)
	}

	return cfg, nil
}
