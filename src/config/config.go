package config

import (
	"fmt"
	"io/ioutil"

	// external deps
	"gopkg.in/yaml.v2"
)

var instance *Config = nil

const configFileLocation = "./config.yaml"

type DatabaseConfig struct {
	Shards       []string `yaml:"shards"`
	ElasticDebug bool
	// drop index on start-up
	DropOnStartup bool `yaml:"drop_on_startup"`
}

type Config struct {
	DatabaseConfig DatabaseConfig `yaml:"database_config"`
	Listen         string         `yaml:"listen"`
}

// verify loaded config -- filter out evident errors
func (c *Config) Verify() error {

	if len(c.DatabaseConfig.Shards) <= 0 {
		return fmt.Errorf("config: verify error: No shards specified")
	}

	return nil
}

// Read configuration from file (path=configFileLocation)
func Read() (*Config, error) {

	// read config file
	bytes, err := ioutil.ReadFile(configFileLocation)
	if err != nil {
		return nil, err
	}

	// decode it
	var result Config
	err = yaml.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}

	// verify values
	err = result.Verify()
	if err != nil {
		return nil, err
	}

	instance = &result
	return instance, nil
}

// Get (prev loaded) configuration
func Get() *Config {
	return instance
}
