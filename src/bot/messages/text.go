package messages

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/textproto"
	"os"
	"time"

	"connect-companion/bot/client"
	"connect-companion/bot/requests"
	"connect-companion/config"
	"connect-companion/database"
	"connect-companion/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MessageType int

const (
	MESSAGE_TEXT                    MessageType = 1
	MESSAGE_CALL_START_TREATMENT    MessageType = 20
	MESSAGE_CALL_START_NO_TREATMENT MessageType = 21
	MESSAGE_FILE                    MessageType = 70
	MESSAGE_TREATMENT_START_BY_USER MessageType = 80
	MESSAGE_TREATMENT_START_BY_SPEC MessageType = 81
	MESSAGE_TREATMENT_CLOSE         MessageType = 82
	MESSAGE_TREATMENT_CLOSE_ACTIVE  MessageType = 90

	MESSAGE_TREATMENT_TO_BOT MessageType = 200
)

type (
	Message struct {
		LineId uuid.UUID `json:"line_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		UserId uuid.UUID `json:"user_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`

		MessageID     uuid.UUID   `json:"message_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		MessageType   MessageType `json:"message_type" binding:"required" example:"1"`
		MessageAuthor *uuid.UUID  `json:"author_id" binding:"omitempty" example:"4e48509f-6366-4897-9544-46f006e47074"`
		MessageTime   string      `json:"message_time" binding:"required" example:"1"`
		Text          string      `json:"text" example:"Привет"`
		Data          struct {
			Redirect string `json:"redirect"`
		} `json:"data"`
	}
)

func (msg *Message) checkError(err error, nextState database.ChatState) (database.ChatState, error) {
	if err != nil {
		logger.Warning("Get error while send message to line", msg.LineId, "for user", msg.UserId, "with error", err)
		return database.STATE_GREETINGS, err
	}
	return nextState, nil
}

func (msg *Message) Start(nextState database.ChatState) (database.ChatState, error) {
	data := requests.DropKeyboardRequest{
		LineID: msg.LineId,
		UserId: msg.UserId,
	}

	jsonData, err := json.Marshal(data)

	_, err = client.Invoke("POST", "/line/drop/keyboard/", "application/json", jsonData)

	return msg.checkError(err, nextState)
}

func (msg *Message) Send(c *gin.Context, text string, nextState database.ChatState, keyboard *[][]requests.KeyboardKey) (database.ChatState, error) {
	cnf := c.MustGet("cnf").(*config.Conf)

	data := requests.MessageRequest{
		LineID:   msg.LineId,
		UserId:   msg.UserId,
		AuthorID: cnf.SpecID,
		Text:     text,
		Keyboard: keyboard,
	}

	jsonData, err := json.Marshal(data)

	_, err = client.Invoke("POST", "/line/send/message/", "application/json", jsonData)

	return msg.checkError(err, nextState)
}

func (msg *Message) RerouteTreatment(c *gin.Context, text string, nextState database.ChatState) (database.ChatState, error) {
	if text != "" {
		_, _ = msg.Send(c, text, nextState, nil)

		time.Sleep(500 * time.Millisecond)
	}

	data := requests.TreatmentRequest{
		LineID: msg.LineId,
		UserId: msg.UserId,
	}

	jsonData, err := json.Marshal(data)

	_, err = client.Invoke("POST", "/line/appoint/start/", "application/json", jsonData)

	return msg.checkError(err, nextState)
}

func (msg *Message) CloseTreatment(c *gin.Context, text string, nextState database.ChatState) (database.ChatState, error) {
	_, _ = msg.Send(c, text, nextState, nil)

	time.Sleep(500 * time.Millisecond)

	data := requests.TreatmentRequest{
		LineID: msg.LineId,
		UserId: msg.UserId,
	}

	jsonData, err := json.Marshal(data)

	_, err = client.Invoke("POST", "/line/drop/treatment/", "application/json", jsonData)

	return msg.checkError(err, nextState)
}

func (msg *Message) StartAndReroute(nextState database.ChatState) (database.ChatState, error) {
	_, err := msg.Start(nextState)
	data := requests.TreatmentRequest{
		LineID: msg.LineId,
		UserId: msg.UserId,
	}

	jsonData, err := json.Marshal(data)

	_, err = client.Invoke("POST", "/line/appoint/start/", "application/json", jsonData)

	return msg.checkError(err, nextState)

}

func (msg *Message) SendFile(c *gin.Context, isImage bool, fileName string, filepath string, comment *string, nextState database.ChatState, keyboard *[][]requests.KeyboardKey) (database.ChatState, error) {
	cnf := c.MustGet("cnf").(*config.Conf)

	data := requests.FileRequest{
		LineID:   msg.LineId,
		UserId:   msg.UserId,
		AuthorID: cnf.SpecID,
		FileName: fileName,
		Comment:  comment,
		Keyboard: keyboard,
	}

	jsonData, err := json.Marshal(data)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	metaPartHeader := textproto.MIMEHeader{}
	metaPartHeader.Set("Content-Disposition", `form-data; name="meta"`)
	metaPartHeader.Set("Content-Type", "application/json")
	metaPart, err := writer.CreatePart(metaPartHeader)
	if err != nil {
		return msg.checkError(err, nextState)
	}
	_, _ = metaPart.Write(jsonData)

	file, err := os.Open(filepath)
	if err != nil {
		return msg.checkError(err, nextState)
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return msg.checkError(err, nextState)
	}
	fi, err := file.Stat()
	if err != nil {
		return msg.checkError(err, nextState)
	}
	_ = file.Close()

	filePart, err := writer.CreateFormFile("file", fi.Name())
	if err != nil {
		return msg.checkError(err, nextState)
	}
	_, _ = filePart.Write(fileContents)

	err = writer.Close()
	if err != nil {
		return msg.checkError(err, nextState)
	}

	if isImage {
		_, err = client.Invoke("POST", "/line/send/image/", writer.FormDataContentType(), body.Bytes())
		return msg.checkError(err, nextState)
	}
	_, err = client.Invoke("POST", "/line/send/file/", writer.FormDataContentType(), body.Bytes())
	return msg.checkError(err, nextState)
}
