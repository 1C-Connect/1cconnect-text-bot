package bot

import (
	"connect-text-bot/internal/config"
	"connect-text-bot/internal/connect/client"
	"connect-text-bot/internal/logger"

	"github.com/gin-gonic/gin"
)

const eventUri = "/connect-push/receive/"

func InitHooks(app *gin.Engine, cnf *config.Conf) {
	logger.Info("Init receiving endpoint...")

	app.POST(eventUri, Receive)

	logger.Info("Setup hooks on 1C-Connect...")

	var err error
	for _, lineID := range cnf.Line {
		logger.Info("- hook for line", lineID)
		connect := client.New(lineID, cnf.ConnectServer.Addr, cnf.Connect.Login, cnf.Connect.Password, cnf.GeneralSettings, cnf.SpecID)

		_, err = connect.SetHook(cnf.Server.Host + eventUri)
		if err != nil {
			logger.Crit("Error while setup hook:", err)
		}

		botsConnect[lineID] = Bot{
			connect: connect,
		}
	}
}

func DestroyHooks() {
	logger.Info("Destroy hooks on 1C-Connect...")

	var err error
	for line_id, b := range botsConnect {
		_, err = b.connect.DeleteHook()
		if err != nil {
			logger.Warning("Error while delete hook:", err)
		}

		delete(botsConnect, line_id)
	}
}
