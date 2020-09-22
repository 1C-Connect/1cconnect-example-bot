package bot

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"connect-companion/bot/messages"
	"connect-companion/bot/requests"
	"connect-companion/config"
	"connect-companion/database"
	"connect-companion/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v7"
)

const (
	BOT_PHRASE_GREETING     = "Выберите, какая информация вас интересует:"
	BOT_PHRASE_SORRY        = "Извините, но я вас не понимаю. Выберите, пожалуйста, один из вариантов:"
	BOT_PHRASE_FILE_SENDING = "Сейчас пришлю соотвествующий файл, подождите."
	BOT_PHRASE_FILE_SENDED  = "Вот, пожалуйста."
	BOT_PHRASE_AGAIN        = "Могу ли я чем-то помочь еще?"
	BOT_PHRASE_RETOUTING    = "Сейчас переведу, секундочку."
	BOT_PHRASE_BYE          = "Спасибо за обращение!"
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

	// Реагируем только на сообщения пользователя
	if (msg.MessageType == messages.MESSAGE_TEXT || msg.MessageType == messages.MESSAGE_FILE) && msg.MessageAuthor != nil && msg.UserId != *msg.MessageAuthor {
		c.Status(http.StatusOK)
		return
	}

	cCp := c.Copy()
	go func(cCp *gin.Context, msg messages.Message) {
		chatState := getState(c, &msg)

		newState, err := processMessage(&msg, &chatState)
		if err != nil {
			logger.Warning("Error processMessage", err)
		}

		err = changeState(c, &msg, &chatState, newState)
		if err != nil {
			logger.Warning("Error changeState", err)
		}
	}(cCp, msg)

	c.Status(http.StatusOK)
}

func getState(c *gin.Context, msg *messages.Message) database.Chat {
	db := c.MustGet("db").(*redis.Client)

	var chatState database.Chat

	dbStateKey := database.PREFIX_STATE + msg.UserId.String() + ":" + msg.LineId.String()

	dbStateRaw, err := db.Get(dbStateKey).Bytes()
	if err == redis.Nil {
		logger.Info("No state in db for " + msg.UserId.String() + ":" + msg.LineId.String())

		chatState = database.Chat{
			PreviousState: database.STATE_GREETINGS,
			CurrentState:  database.STATE_GREETINGS,
		}
	} else if err != nil {
		logger.Warning("Error while reading state from redis", err)
	} else {
		err = json.Unmarshal(dbStateRaw, &chatState)
		if err != nil {
			logger.Warning("Error while decoding state", err)
		}
	}

	return chatState
}

func changeState(c *gin.Context, msg *messages.Message, chatState *database.Chat, toState database.ChatState) error {
	db := c.MustGet("db").(*redis.Client)

	chatState.PreviousState = chatState.CurrentState
	chatState.CurrentState = toState

	data, err := json.Marshal(chatState)
	if err != nil {
		logger.Warning("Error while change state to db", err)

		return err
	}

	dbStateKey := database.PREFIX_STATE + msg.UserId.String() + ":" + msg.LineId.String()

	result, err := db.Set(dbStateKey, data, database.EXPIRE).Result()
	logger.Debug("Write state to db result", result)
	if err != nil {
		logger.Warning("Error while write state to db", err)
	}

	return nil
}

func checkErrorForSend(msg *messages.Message, err error, nextState database.ChatState) (database.ChatState, error) {
	if err != nil {
		logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
		return database.STATE_GREETINGS, err
	}

	return nextState, nil
}

func processMessage(msg *messages.Message, chatState *database.Chat) (database.ChatState, error) {
	switch msg.MessageType {
	case messages.MESSAGE_TREATMENT_START_BY_USER:
		return chatState.CurrentState, nil
	case messages.MESSAGE_CALL_START_TREATMENT,
		messages.MESSAGE_CALL_START_NO_TREATMENT,
		messages.MESSAGE_TREATMENT_START_BY_SPEC,
		messages.MESSAGE_TREATMENT_CLOSE,
		messages.MESSAGE_TREATMENT_CLOSE_ACTIVE:
		_, err := HideKeyboard(msg.LineId, msg.UserId)

		return checkErrorForSend(msg, err, database.STATE_GREETINGS)
	case messages.MESSAGE_TEXT:
		keyboardMain := &[][]requests.KeyboardKey{
			{{Id: "1", Text: "Памятка сотрудника"}},
			{{Id: "2", Text: "Положение о персонале"}},
			{{Id: "3", Text: "Регламент о пожеланиях"}},
			{{Id: "9", Text: "Закрыть обращение"}},
			{{Id: "0", Text: "Перевести на специалиста"}},
		}
		keyboardParting := &[][]requests.KeyboardKey{
			{{Id: "1", Text: "Да"}, {Id: "2", Text: "Нет"}},
			{{Id: "0", Text: "Перевести на специалиста"}},
		}

		switch chatState.CurrentState {
		case database.STATE_DUMMY, database.STATE_GREETINGS:
			_, err := SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_GREETING, keyboardMain)

			return checkErrorForSend(msg, err, database.STATE_MAIN_MENU)
		case database.STATE_MAIN_MENU:
			comment := BOT_PHRASE_FILE_SENDED
			switch strings.ToLower(strings.TrimSpace(msg.Text)) {
			case "1", "памятка сотрудника":
				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_FILE_SENDING, nil)

				filePath, _ := filepath.Abs(filepath.Join(cnf.FilesDir, "Памятка сотрудника.pdf"))
				_, err := SendFile(false, msg.LineId, msg.UserId, "Памятка сотрудника.pdf", filePath, &comment, nil)

				time.Sleep(3 * time.Second)

				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_AGAIN, keyboardParting)

				return checkErrorForSend(msg, err, database.STATE_PARTING)
			case "2", "положение о персонале":
				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_FILE_SENDING, nil)

				filePath, _ := filepath.Abs(filepath.Join(cnf.FilesDir, "Положение о персонале.pdf"))
				_, err := SendFile(false, msg.LineId, msg.UserId, "Положение о персонале.pdf", filePath, &comment, nil)

				time.Sleep(3 * time.Second)

				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_AGAIN, keyboardParting)

				return checkErrorForSend(msg, err, database.STATE_PARTING)
			case "3", "регламент о пожеланиях":
				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_FILE_SENDING, nil)

				filePath, _ := filepath.Abs(filepath.Join(cnf.FilesDir, "Регламент.pdf"))
				_, err := SendFile(false, msg.LineId, msg.UserId, "Регламент.pdf", filePath, &comment, nil)

				time.Sleep(3 * time.Second)

				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_AGAIN, keyboardParting)

				return checkErrorForSend(msg, err, database.STATE_PARTING)
			case "9", "закрыть обращение":
				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_BYE, nil)

				_, err := CloseTreatment(msg.LineId, msg.UserId)

				return checkErrorForSend(msg, err, database.STATE_GREETINGS)
			case "0", "перевести на специалиста":
				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_RETOUTING, nil)

				_, err := RerouteTreatment(msg.LineId, msg.UserId)

				return checkErrorForSend(msg, err, database.STATE_GREETINGS)
			default:
				_, err := SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_SORRY, keyboardMain)

				return checkErrorForSend(msg, err, database.STATE_MAIN_MENU)
			}
		case database.STATE_PARTING:
			switch strings.ToLower(strings.TrimSpace(msg.Text)) {
			case "1", "да":
				_, err := SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_GREETING, keyboardMain)

				return checkErrorForSend(msg, err, database.STATE_MAIN_MENU)
			case "2", "нет":
				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_BYE, nil)

				time.Sleep(500 * time.Millisecond)

				_, err := CloseTreatment(msg.LineId, msg.UserId)

				return checkErrorForSend(msg, err, database.STATE_GREETINGS)
			case "0", "перевести на специалиста":
				_, _ = SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_RETOUTING, nil)

				time.Sleep(500 * time.Millisecond)

				_, err := RerouteTreatment(msg.LineId, msg.UserId)

				return checkErrorForSend(msg, err, database.STATE_GREETINGS)
			default:
				_, err := SendMessage(msg.LineId, msg.UserId, BOT_PHRASE_SORRY, keyboardParting)

				return checkErrorForSend(msg, err, database.STATE_PARTING)
			}
		}
	case messages.MESSAGE_FILE:
		_, err := HideKeyboard(msg.LineId, msg.UserId)
		_, err = RerouteTreatment(msg.LineId, msg.UserId)

		return checkErrorForSend(msg, err, database.STATE_GREETINGS)
	}

	return database.STATE_DUMMY, errors.New("I don't know hat i mus do!")
}
