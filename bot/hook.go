package bot

import (
	"connect-text-bot/bot/client"
	"connect-text-bot/internal/config"
	"connect-text-bot/internal/logger"

	"github.com/gin-gonic/gin"
)

func InitHooks(app *gin.Engine, cnf *config.Conf) {
	logger.Info("Init receiving endpoint...")

	app.POST("/connect-push/receive/", Receive)

	logger.Info("Setup hooks on 1C-Connect...")

	for i := range cnf.Line {
		logger.Info("- hook for line", cnf.Line[i])

		_, err := client.SetHook(cnf, cnf.Line[i])
		if err != nil {
			logger.Crit("Error while setup hook:", err)
		}
	}
}

func DestroyHooks(cnf *config.Conf) {
	logger.Info("Destroy hooks on 1C-Connect...")

	for i := range cnf.Line {
		_, err := client.DeleteHook(cnf, cnf.Line[i])
		if err != nil {
			logger.Warning("Error while delete hook:", err)
		}
	}
}
