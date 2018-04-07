package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

const (
	ConfigFilename = "./cli-config.json"
)

var config *Config // The single instance of config

type Config struct {
	Host                 string `json:"Host"`
	DestroyAddress       string `json:"DestroyAddress"`
}

func (config *Config) readConfigFile() error {
	data, err := ioutil.ReadFile(ConfigFilename)
	if err != nil {
		return err
	}
	// Remove the UTF-8 Byte Order Mark
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	err = json.Unmarshal(data, config)
	if err != nil {
		return err
	}
	return nil
}

func Params() *Config {
	if config == nil {
		config = &Config{
			"localhost:20336",
			"",
		}
		err := config.readConfigFile()
		if err != nil {
			fmt.Println("Read config file error:", err)
		}
	}
	return config
}
