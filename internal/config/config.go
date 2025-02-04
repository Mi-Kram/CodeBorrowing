package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

type Config struct {
	Logs           string
	Storage        string
	CheckerPath    string
	StorageSize    uint64
	MainServerHost string
	MainServerKey  string
}

const (
	envLogs           = "logs"
	envStorage        = "storage"
	envStorageSize    = "storageSize"
	envCrossCheckLib  = "checkerPath"
	envMainServerHost = "mainServerHost"
	envMainServerKey  = "mainServerKey"
)

var instance *Config
var once = sync.Once{}

func GetConfig() (*Config, error) {
	var configErr error = nil
	isErr := true

	once.Do(func() {
		instance = &Config{}
		cacheSize, err := strconv.ParseUint(os.Getenv(envStorageSize), 10, 64)
		if err != nil {
			configErr = err
			return
		}

		instance.Logs = os.Getenv(envLogs)
		instance.Storage = os.Getenv(envStorage)
		instance.CheckerPath = os.Getenv(envCrossCheckLib)
		instance.MainServerHost = os.Getenv(envMainServerHost)
		instance.MainServerKey = os.Getenv(envMainServerKey)
		instance.StorageSize = cacheSize

		if instance.Logs == "" {
			err = fmt.Errorf("environment variable: \"%s\" not found", envLogs)
		} else if instance.Storage == "" {
			err = fmt.Errorf("environment variable: \"%s\" not found", envStorage)
		} else if instance.CheckerPath == "" {
			err = fmt.Errorf("environment variable: \"%s\" not found", envCrossCheckLib)
		} else {
			isErr = false
		}
	})

	if isErr {
		return instance, configErr
	}
	return instance, nil
}
