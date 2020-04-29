package main

/**
This module contains configuration related types and logic.
*/
import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	LogLevel    string        `ini:"log_level"`
	Interval    time.Duration `ini:"interval"`
	*AuthConfig `ini:"auth"`
}
type AuthConfig struct {
	Username string `ini:"username"`
	Password string `ini:"password"`
	RouterIP string `ini:"router_ip"`
}

func LoadConfig() (c Config, err error) {
	var userHomeDir string
	if userHomeDir, err = os.UserHomeDir(); err != nil {
		return
	}
	if err = ini.MapTo(&c, filepath.Join(userHomeDir, `.config`, `otecstar`, `config.ini`)); err != nil {
		return
	}
	if c.AuthConfig == nil {
		err = fmt.Errorf("auth config empty")
	}
	return
}
