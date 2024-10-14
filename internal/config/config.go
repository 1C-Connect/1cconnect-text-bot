package config

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	// configuration contains the application settings
	Conf struct {
		RunInDebug bool `yaml:"debug"`

		Server Server `yaml:"server"`

		Connect Connect `yaml:"connect"`

		FilesDir  string      `yaml:"files_dir"`
		BotConfig string      `yaml:"bot_config"`
		SpecID    *uuid.UUID  `yaml:"spec_id"`
		Line      []uuid.UUID `yaml:"line"`
	}

	Server struct {
		Host   string `yaml:"host"`
		Listen string `yaml:"listen"`
	}

	Connect struct {
		Server     string `yaml:"server"`
		SoapServer string `yaml:"soap_server"`
		Login      string `yaml:"login"`
		Password   string `yaml:"password"`
	}
)

func Inject(key string, cnf *Conf) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(key, cnf)
	}
}
