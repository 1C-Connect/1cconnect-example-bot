package config

import (
	"connect-companion/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	// configuration contains the application settings
	Conf struct {
		RunInDebug bool `yaml:"debug"`

		Server   Server         `yaml:"server"`
		Database database.Redis `yaml:"database"`

		Connect Connect `yaml:"connect"`

		FilesDir string      `yaml:"files_dir"`
		Line     []uuid.UUID `yaml:"line"`
	}

	Server struct {
		Host   string `yaml:"host"`
		Listen string `yaml:"listen"`
	}

	Connect struct {
		Server   string `yaml:"server"`
		Login    string `yaml:"login"`
		Password string `yaml:"password"`
	}
)

func Inject(cnf *Conf) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("cnf", cnf)
	}
}
