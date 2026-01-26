package config

import "time"

type HTTPConfig struct {
	Host string `yaml:"host" env:"HOST"`
	Port int    `yaml:"port" env:"PORT"`

	Timeout time.Duration `yaml:"timeout" env:"TIMEOUT"`
	RPS     int           `yaml:"rps" env:"RPS"`
}

type NATSConfig struct {
	URL            string        `yaml:"url" env:"NATS_URL"`
	RequestTimeout time.Duration `yaml:"request_timeout" env:"NATS_REQUEST_TIMEOUT"`
	MaxReconnects  int           `yaml:"max_reconnects" env:"NATS_MAX_RECONNECTS"`
	ReconnectWait  time.Duration `yaml:"reconnect_wait" env:"NATS_RECONNECT_WAIT"`
}

type Config struct {
	HTTP HTTPConfig `yaml:"http"`
	NATS NATSConfig `yaml:"nats"`
}
