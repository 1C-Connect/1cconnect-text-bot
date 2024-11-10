package config

import (
	"connect-text-bot/internal/us"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	// configuration contains the application settings
	Conf struct {
		Server Server `yaml:"server"`

		Connect Connect `yaml:"connect"`

		ConnectServer ConnectServer `yaml:"connect_server"`
		UsServer      us.UsServer   `yaml:"us_server"`

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
		Login    string `yaml:"login"`
		Password string `yaml:"password"`
	}

	ConnectServer struct {
		Addr string `yaml:"addr"`
	}
)

func Inject(key string, cnf *Conf) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(key, cnf)
	}
}
