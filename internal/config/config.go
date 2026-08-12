package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	DefaultInstance string              `toml:"default_instance"`
	Instances       map[string]Instance `toml:"instances"`
}

type Instance struct {
	BaseURL             string   `toml:"base_url"`
	APIURL              string   `toml:"api_url"`
	Credential          string   `toml:"credential"`
	AllowedRepositories []string `toml:"allowed_repositories"`
	ReadOnly            bool     `toml:"read_only"`
	TLSCAFile           string   `toml:"tls_ca_file"`
}

type NamedInstance struct {
	Alias string
	Instance
}

func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("configuration file does not exist: %s", path)
		}
		return nil, fmt.Errorf("read configuration: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse configuration: %w", err)
	}
	if cfg.Instances == nil || len(cfg.Instances) == 0 {
		return nil, errors.New("configuration contains no instances")
	}
	for alias, instance := range cfg.Instances {
		normalized, err := normalize(instance)
		if err != nil {
			return nil, fmt.Errorf("instance %q: %w", alias, err)
		}
		cfg.Instances[alias] = normalized
	}
	if cfg.DefaultInstance != "" {
		if _, ok := cfg.Instances[cfg.DefaultInstance]; !ok {
			return nil, fmt.Errorf("default instance %q is not defined", cfg.DefaultInstance)
		}
	}
	return &cfg, nil
}

func DefaultPath() (string, error) {
	if path := os.Getenv("FORGEJO_BRIDGE_CONFIG"); path != "" {
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, "forgejo-bridge", "config.toml"), nil
}

func (c *Config) Select(alias string) (NamedInstance, error) {
	if alias == "" {
		alias = os.Getenv("FORGEJO_BRIDGE_INSTANCE")
	}
	if alias == "" {
		alias = c.DefaultInstance
	}
	if alias == "" && len(c.Instances) == 1 {
		for candidate := range c.Instances {
			alias = candidate
		}
	}
	if alias == "" {
		return NamedInstance{}, errors.New("instance is ambiguous; select one explicitly")
	}
	instance, ok := c.Instances[alias]
	if !ok {
		return NamedInstance{}, fmt.Errorf("instance %q is not configured", alias)
	}
	return NamedInstance{Alias: alias, Instance: instance}, nil
}

func (c *Config) Aliases() []string {
	aliases := make([]string, 0, len(c.Instances))
	for alias := range c.Instances {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func normalize(instance Instance) (Instance, error) {
	if instance.BaseURL == "" {
		return Instance{}, errors.New("base_url is required")
	}
	base, err := url.Parse(instance.BaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return Instance{}, errors.New("base_url must be an absolute http or https URL")
	}
	if base.Scheme != "https" && base.Scheme != "http" {
		return Instance{}, errors.New("base_url must use http or https")
	}
	if base.RawQuery != "" || base.Fragment != "" || base.User != nil {
		return Instance{}, errors.New("base_url must not contain credentials, query, or fragment")
	}
	instance.BaseURL = strings.TrimRight(base.String(), "/")
	if instance.APIURL == "" {
		instance.APIURL = instance.BaseURL + "/api/v1"
	}
	api, err := url.Parse(instance.APIURL)
	if err != nil || api.Scheme == "" || api.Host == "" {
		return Instance{}, errors.New("api_url must be an absolute URL")
	}
	if !strings.EqualFold(api.Host, base.Host) {
		return Instance{}, errors.New("api_url must use the same origin as base_url")
	}
	instance.APIURL = strings.TrimRight(api.String(), "/")
	sort.Strings(instance.AllowedRepositories)
	return instance, nil
}

func (i Instance) RepositoryAllowed(fullName string) bool {
	if len(i.AllowedRepositories) == 0 {
		return true
	}
	for _, allowed := range i.AllowedRepositories {
		if strings.EqualFold(strings.TrimSpace(allowed), fullName) {
			return true
		}
	}
	return false
}

func (i Instance) ResolveCredential() (string, error) {
	if i.Credential == "" {
		return "", nil
	}
	scheme, value, ok := strings.Cut(i.Credential, ":")
	if !ok || value == "" {
		return "", errors.New("credential must use a supported scheme")
	}
	switch scheme {
	case "env":
		token := os.Getenv(value)
		if token == "" {
			return "", fmt.Errorf("credential environment variable %q is empty", value)
		}
		return strings.TrimSpace(token), nil
	case "file":
		raw, err := os.ReadFile(value)
		if err != nil {
			return "", fmt.Errorf("read credential file: %w", err)
		}
		token := strings.TrimSpace(string(raw))
		if token == "" {
			return "", errors.New("credential file is empty")
		}
		return token, nil
	default:
		return "", fmt.Errorf("unsupported credential scheme %q", scheme)
	}
}
