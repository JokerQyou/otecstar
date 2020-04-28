package main

/**
This module contains configuration related types and logic.
*/
import (
	"fmt"
	"gopkg.in/ini.v1"
)

type Config struct {
	LogLevel    string `ini:"log_level"`
	*AuthConfig `ini:"auth"`
}
type AuthConfig struct {
	Username string `ini:"username"`
	Password string `ini:"password"`
	RouterIP string `ini:"router_ip"`
}

func LoadConfig() (c Config, err error) {
	if err = ini.MapTo(&c, "./config.ini"); err != nil {
		return
	}
	if c.AuthConfig == nil {
		err = fmt.Errorf("auth config empty")
	}
	return
}
