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

	go func(msg messages.Message) {
		if msg.MessageType == messages.MESSAGE_TEXT {
			switch strings.ToLower(msg.Text) {
			case "здрасти!", "\\привет":
				keyboard := &[][]requests.KeyboardKey{
					{{Id: "\\назад", Text: "Назад"}},
				}

				_, err := SendMessage(msg.LineId, msg.UserId, "Хохо печениги на месте!", keyboard)
				if err != nil {
					logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			case "кинь файлом", "\\файл":
				keyboard := &[][]requests.KeyboardKey{
					{{Id: "\\назад", Text: "Назад"}},
				}
				filePath, _ := filepath.Abs("build/image.jpg")
				_, err := SendFile(msg.LineId, msg.UserId, "Мальчик.jpg", filePath, keyboard)
				if err != nil {
					logger.Warning("Get error while send file to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			case "закрыть обращение", "\\закрыть":
				_, err := CloseTreatment(msg.LineId, msg.UserId)
				if err != nil {
					logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			case "переведи", "\\перевод":
				_, err := RerouteTreatment(msg.LineId, msg.UserId)
				if err != nil {
					logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			default:
				keyboard := &[][]requests.KeyboardKey{
					{{Id: "\\привет", Text: "Здрасти!"}, {Id: "\\файл", Text: "Кинь файлом"}},
					{{Id: "\\закрыть", Text: "Закрыть обращение"}, {Id: "\\перевод", Text: "Переведи"}},
				}

				_, err := SendMessage(msg.LineId, msg.UserId, "Привет я демо бот который умеет следующее:", keyboard)
				if err != nil {
					logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
				}
			}

		}

		if msg.MessageType == messages.MESSAGE_TREATMENT_CLOSE {
			_, err := HideKeyboard(msg.LineId, msg.UserId)
			if err != nil {
				logger.Warning("Get error while hide keyboard to line", msg.LineId, "for user", msg.UserId, "with error", err)
			}
		}
	}(msg)

	c.Status(http.StatusOK)
}
