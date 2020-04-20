package bot

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"сonnect-companion/logger"
)

func InitHooks(app *gin.Engine, lines []uuid.UUID) {
	logger.Info("Init receiving endpoint...")

	app.POST("/connect-push/receive/", Receive)

	logger.Info("Setup hooks on 1C-Connect...")

	for i := range lines {
		logger.Info("- hook for line", lines[i])

		_, err := setHook(&lines[i])
		if err != nil {
			logger.Warning("Error while setup hook:", err)
		}
	}
}

func DestroyHooks(lines []uuid.UUID) {
	logger.Info("Destroy hooks on 1C-Connect...")

	for i := range lines {
		_, err := deleteHook(&lines[i])
		if err != nil {
			logger.Warning("Error while delete hook:", err)
		}
	}
}
