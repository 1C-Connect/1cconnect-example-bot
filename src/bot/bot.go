package bot

import (
	"net/http"
	"path/filepath"
	"strings"

	"сonnect-companion/bot/messages"
	"сonnect-companion/bot/requests"
	"сonnect-companion/config"
	"сonnect-companion/logger"

	"github.com/gin-gonic/gin"
)

var (
	cnf = &config.Conf{}
)

func Configure(c *config.Conf) {
	cnf = c
}

func Receive(c *gin.Context) {
	var msg messages.Message
	if err := c.BindJSON(&msg); err != nil {
		logger.Warning("Error while receive message", err)

		c.Status(http.StatusBadRequest)
		return
	}

	logger.Debug("Receive message:", msg)

	go func(msg messages.Message) {
		switch msg.MessageType {
		case messages.MESSAGE_TEXT:
			switch strings.ToLower(msg.Text) {
			case "здрасти!", "1", "1)":
				keyboard := &[][]requests.KeyboardKey{
					{{Id: "b", Text: "Назад"}},
				}

				_, err := SendMessage(msg.LineId, msg.UserId, "Хохо печениги на месте!", keyboard)
				if err != nil {
					logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			case "кинь файлом", "2", "2)":
				keyboard := &[][]requests.KeyboardKey{
					{{Id: "b", Text: "Назад"}},
				}
				filePath, _ := filepath.Abs("build/image.jpg")
				_, err := SendFile(msg.LineId, msg.UserId, "Мальчик.jpg", filePath, keyboard)
				if err != nil {
					logger.Warning("Get error while send file to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			case "закрыть обращение", "3", "3)":
				_, err := CloseTreatment(msg.LineId, msg.UserId)
				if err != nil {
					logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			case "переведи", "4", "4)":
				_, err := RerouteTreatment(msg.LineId, msg.UserId)
				if err != nil {
					logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			default:
				keyboard := &[][]requests.KeyboardKey{
					{{Id: "1", Text: "Здрасти!"}, {Id: "2", Text: "Кинь файлом"}},
					{{Id: "3", Text: "Закрыть обращение"}, {Id: "4", Text: "Переведи"}},
				}

				_, err := SendMessage(msg.LineId, msg.UserId, "Привет я демо бот который умеет следующее:", keyboard)
				if err != nil {
					logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			}
		case messages.MESSAGE_TREATMENT_CLOSE,
			messages.MESSAGE_TREATMENT_CLOSE_ACTIVE,
			messages.MESSAGE_TREATMENT_CLOSE_DEL_LINE,
			messages.MESSAGE_TREATMENT_CLOSE_DEL_SUBS,
			messages.MESSAGE_TREATMENT_CLOSE_DEL_USER:
			_, err := HideKeyboard(msg.LineId, msg.UserId)
			if err != nil {
				logger.Warning("Get error while hide keyboard to line", msg.LineId, "for user", msg.UserId, "with error", err)
			}
		}
	}(msg)

	c.Status(http.StatusOK)
}
