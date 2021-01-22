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

		newState, err := processMessage(c, &msg, &chatState)
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

func processMessage(c *gin.Context, msg *messages.Message, chatState *database.Chat) (database.ChatState, error) {
	cnf := c.MustGet("cnf").(*config.Conf)

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

	switch msg.MessageType {
	case messages.MESSAGE_TREATMENT_START_BY_USER:
		return chatState.CurrentState, nil
	case messages.MESSAGE_CALL_START_TREATMENT,
		messages.MESSAGE_CALL_START_NO_TREATMENT,
		messages.MESSAGE_TREATMENT_START_BY_SPEC,
		messages.MESSAGE_TREATMENT_CLOSE,
		messages.MESSAGE_TREATMENT_CLOSE_ACTIVE:

		return msg.Start(database.STATE_GREETINGS)
	case messages.MESSAGE_TREATMENT_TO_BOT:
		// Спец перевел на бота. Смотрим куда именно
		switch msg.Data.Redirect {
		case "add_collegue,level:1":
			// return msg.Send("Как добавить сотрудника в 1с-коннект?\n"+BOT_PHRASE_DEMO_0, database.STATE_DEMO_1, keyboardDemo1)
			return msg.Send(c, BOT_PHRASE_GREETING, database.STATE_MAIN_MENU, keyboardMain)
		case "add_collegue,level:3":
			// filePath, _ := filepath.Abs(filepath.Join(cnf.FilesDir, "manage_spec.png"))
			// return msg.SendFile(true, "manage_spec.png", filePath, BOT_PHRASE_DEMO_2, database.STATE_DEMO_3, keyboardDemo3)
			return msg.Send(c, BOT_PHRASE_GREETING, database.STATE_MAIN_MENU, keyboardMain)
		default:
			return msg.Send(c, BOT_PHRASE_GREETING, database.STATE_MAIN_MENU, keyboardMain)
		}
	case messages.MESSAGE_TEXT:
		text := strings.ToLower(strings.TrimSpace(msg.Text))

		switch chatState.CurrentState {
		case database.STATE_DUMMY, database.STATE_GREETINGS:
			return msg.Send(c, BOT_PHRASE_GREETING, database.STATE_MAIN_MENU, keyboardMain)
		case database.STATE_MAIN_MENU:
			comment := BOT_PHRASE_FILE_SENDED
			switch text {
			case "1", "памятка сотрудника":
				msg.Send(c, BOT_PHRASE_FILE_SENDING, database.STATE_PARTING, nil)

				filePath, _ := filepath.Abs(filepath.Join(cnf.FilesDir, "Памятка сотрудника.pdf"))
				msg.SendFile(c, false, "Памятка сотрудника.pdf", filePath, &comment, database.STATE_PARTING, nil)

				time.Sleep(3 * time.Second)

				return msg.Send(c, BOT_PHRASE_AGAIN, database.STATE_PARTING, keyboardParting)
			case "2", "положение о персонале":
				msg.Send(c, BOT_PHRASE_FILE_SENDING, database.STATE_PARTING, nil)

				filePath, _ := filepath.Abs(filepath.Join(cnf.FilesDir, "Положение о персонале.pdf"))
				msg.SendFile(c, false, "Положение о персонале.pdf", filePath, &comment, database.STATE_PARTING, nil)

				time.Sleep(3 * time.Second)

				return msg.Send(c, BOT_PHRASE_AGAIN, database.STATE_PARTING, keyboardParting)
			case "3", "регламент о пожеланиях":
				msg.Send(c, BOT_PHRASE_FILE_SENDING, database.STATE_PARTING, nil)

				filePath, _ := filepath.Abs(filepath.Join(cnf.FilesDir, "Регламент.pdf"))
				msg.SendFile(c, false, "Регламент.pdf", filePath, &comment, database.STATE_PARTING, nil)

				time.Sleep(3 * time.Second)

				return msg.Send(c, BOT_PHRASE_AGAIN, database.STATE_PARTING, keyboardParting)
			case "9", "закрыть обращение":
				return msg.CloseTreatment(c, BOT_PHRASE_BYE, database.STATE_GREETINGS)
			case "0", "перевести на специалиста":
				return msg.RerouteTreatment(c, BOT_PHRASE_RETOUTING, database.STATE_GREETINGS)
			default:
				return msg.Send(c, BOT_PHRASE_SORRY, database.STATE_MAIN_MENU, keyboardMain)
			}
		case database.STATE_PARTING:
			switch text {
			case "1", "да":
				return msg.Send(c, BOT_PHRASE_GREETING, database.STATE_MAIN_MENU, keyboardMain)
			case "2", "нет":
				return msg.CloseTreatment(c, BOT_PHRASE_BYE, database.STATE_GREETINGS)
			case "0", "перевести на специалиста":
				return msg.RerouteTreatment(c, BOT_PHRASE_RETOUTING, database.STATE_GREETINGS)
			default:
				return msg.Send(c, BOT_PHRASE_SORRY, database.STATE_PARTING, keyboardParting)
			}
		}
	case messages.MESSAGE_FILE:
		return msg.StartAndReroute(database.STATE_GREETINGS)
	}

	return database.STATE_DUMMY, errors.New("I don't know hat i mus do!")
}
