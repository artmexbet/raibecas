package config

import "time"

type HTTPConfig struct {
	Host string `yaml:"host" env:"HOST"`
	Port int    `yaml:"port" env:"PORT"`

	Timeout time.Duration `yaml:"timeout" env:"TIMEOUT"`
	RPS     int           `yaml:"rps" env:"RPS"`
}
