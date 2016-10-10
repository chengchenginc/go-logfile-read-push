package config

import (
	_ "bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	_ "os"
)

type TomlConfig struct {
	Title   string
	Redis   RedisConfig   `toml:"redis"`
	LogFile LogFileConfig `toml:"logfile"`
}

type RedisConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

type LogFileConfig struct {
	FilePath string
}

var Config = TomlConfig{}

func LoadConfig(filePath string) error {
	// 如果配置文件不存在
	configBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	} else {
		fmt.Printf("%v", string(configBytes))
	}
	if _, err := toml.DecodeFile(filePath, &Config); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
