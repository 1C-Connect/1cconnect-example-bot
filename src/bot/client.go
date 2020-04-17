package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"сonnect-companion/bot/requests"
	"сonnect-companion/logger"
)

var (
	client = &http.Client{}
)

type (
	HttpError struct {
		Code    int
		Message string
	}
)

func (e *HttpError) Error() string {
	return ""
}

func setHook(lineId *uuid.UUID) (content []byte, err error) {
	reqUrl := fmt.Sprintf("%s/v1/hook/", cnf.Connect.Server)

	connectHookUrl := fmt.Sprintf("%s/connect-push/receive/", cnf.Server.Host)
	data := requests.HookSetupRequest{
		Id:   lineId,
		Type: "bot",
		Url:  connectHookUrl,
	}
	jsonData, err := json.Marshal(data)

	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Warning("Error while setup hook", err)
	}

	return invoke(req, "application/json")
}

func deleteHook(lineId *uuid.UUID) (content []byte, err error) {
	reqUrl := fmt.Sprintf("%s/v1/hook/%s/%s/", cnf.Connect.Server, "bot", lineId)

	req, err := http.NewRequest("DELETE", reqUrl, nil)
	if err != nil {
		logger.Warning("Error while destroy hook", err)
	}

	return invoke(req, "application/json")
}

func SendMessage(lineId uuid.UUID, userId uuid.UUID, text string, keyboard *[][]requests.KeyboardKey) (content []byte, err error) {
	reqUrl := fmt.Sprintf("%s/v1/line/send/message/", cnf.Connect.Server)

	data := requests.MessageRequest{
		LineID:   lineId,
		UserId:   userId,
		Text:     text,
		Keyboard: keyboard,
	}

	jsonData, err := json.Marshal(data)

	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Warning("Error while send message", err)
	}

	return invoke(req, "application/json")
}

func SendFile(lineId uuid.UUID, userId uuid.UUID, fileName string, filepath string, keyboard *[][]requests.KeyboardKey) (content []byte, err error) {
	reqUrl := fmt.Sprintf("%s/v1/line/send/file/", cnf.Connect.Server)

	data := requests.FileRequest{
		LineID:   lineId,
		UserId:   userId,
		FileName: fileName,
		Keyboard: keyboard,
	}

	jsonData, err := json.Marshal(data)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	metaPartHeader := textproto.MIMEHeader{}
	metaPartHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="meta"`))
	metaPartHeader.Set("Content-Type", "application/json")
	metaPart, err := writer.CreatePart(metaPartHeader)
	if err != nil {
		return nil, err
	}
	_, _ = metaPart.Write(jsonData)

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	_ = file.Close()

	filePart, err := writer.CreateFormFile("file", fi.Name())
	if err != nil {
		return nil, err
	}
	_, _ = filePart.Write(fileContents)

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", reqUrl, body)
	if err != nil {
		logger.Warning("Error while send message with file", err)
	}

	return invoke(req, writer.FormDataContentType())
}

func HideKeyboard(lineId uuid.UUID, userId uuid.UUID) (content []byte, err error) {
	reqUrl := fmt.Sprintf("%s/v1/line/drop/keyboard/", cnf.Connect.Server)

	data := requests.DropKeyboardRequest{
		LineID: lineId,
		UserId: userId,
	}

	jsonData, err := json.Marshal(data)

	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Warning("Error while drop keyboard", err)
	}

	return invoke(req, "application/json")
}

func CloseTreatment(lineId uuid.UUID, userId uuid.UUID) (content []byte, err error) {
	reqUrl := fmt.Sprintf("%s/v1/line/drop/treatment/", cnf.Connect.Server)

	data := requests.TreatmentRequest{
		LineID: lineId,
		UserId: userId,
	}

	jsonData, err := json.Marshal(data)

	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Warning("Error while drop keyboard", err)
	}

	return invoke(req, "application/json")
}

func RerouteTreatment(lineId uuid.UUID, userId uuid.UUID) (content []byte, err error) {
	reqUrl := fmt.Sprintf("%s/v1/line/appoint/start/", cnf.Connect.Server)

	data := requests.TreatmentRequest{
		LineID: lineId,
		UserId: userId,
	}

	jsonData, err := json.Marshal(data)

	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Warning("Error while drop keyboard", err)
	}

	return invoke(req, "application/json")
}

func RerouteTreatmentToSpec(lineId uuid.UUID, userId uuid.UUID, specId uuid.UUID) (content []byte, err error) {
	reqUrl := fmt.Sprintf("%s/v1/line/appoint/spec/", cnf.Connect.Server)

	data := requests.TreatmentWithSpecRequest{
		LineID: lineId,
		UserId: userId,
		SpecId: specId,
	}

	jsonData, err := json.Marshal(data)

	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Warning("Error while drop keyboard", err)
	}

	return invoke(req, "application/json")
}

func invoke(req *http.Request, contentType string) (content []byte, err error) {
	req.SetBasicAuth(cnf.Connect.Login, cnf.Connect.Password)
	req.Header.Set("Content-Type", contentType)

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Warning("Error while read response body", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, &HttpError{
				Code:    resp.StatusCode,
				Message: string(bodyBytes),
			}
		}

		return bodyBytes, nil
	}
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
