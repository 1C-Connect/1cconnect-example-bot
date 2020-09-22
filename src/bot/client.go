package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"

	"connect-companion/bot/requests"
	"connect-companion/logger"

	"github.com/google/uuid"
)

var (
	client = &http.Client{}
)

type (
	HttpError struct {
		Url     string
		Code    int
		Message string
	}
)

func (e *HttpError) Error() string {
	return fmt.Sprintf("Http request failed for %s with code %d and message:\n%s", e.Url, e.Code, e.Message)
}

func setHook(lineId uuid.UUID) (content []byte, err error) {
	data := requests.HookSetupRequest{
		Id:   lineId,
		Type: "bot",
		Url:  cnf.Server.Host + "/connect-push/receive/",
	}
	jsonData, err := json.Marshal(data)

	return invoke("POST", "/hook/", "application/json", jsonData)
}

func deleteHook(lineId uuid.UUID) (content []byte, err error) {
	return invoke("DELETE", "/hook/bot/"+lineId.String()+"/", "application/json", nil)
}

func SendMessage(lineId uuid.UUID, userId uuid.UUID, text string, keyboard *[][]requests.KeyboardKey) (content []byte, err error) {
	data := requests.MessageRequest{
		LineID:   lineId,
		UserId:   userId,
		Text:     text,
		Keyboard: keyboard,
	}

	jsonData, err := json.Marshal(data)

	return invoke("POST", "/line/send/message/", "application/json", jsonData)
}

func SendFile(isImage bool, lineId uuid.UUID, userId uuid.UUID, fileName string, filepath string, comment *string, keyboard *[][]requests.KeyboardKey) (content []byte, err error) {
	data := requests.FileRequest{
		LineID:   lineId,
		UserId:   userId,
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

	if isImage {
		return invoke("POST", "/line/send/image/", writer.FormDataContentType(), body.Bytes())
	}
	return invoke("POST", "/line/send/file/", writer.FormDataContentType(), body.Bytes())
}

func HideKeyboard(lineId uuid.UUID, userId uuid.UUID) (content []byte, err error) {
	data := requests.DropKeyboardRequest{
		LineID: lineId,
		UserId: userId,
	}

	jsonData, err := json.Marshal(data)

	return invoke("POST", "/line/drop/keyboard/", "application/json", jsonData)
}

func CloseTreatment(lineId uuid.UUID, userId uuid.UUID) (content []byte, err error) {
	data := requests.TreatmentRequest{
		LineID: lineId,
		UserId: userId,
	}

	jsonData, err := json.Marshal(data)

	return invoke("POST", "/line/drop/treatment/", "application/json", jsonData)
}

func RerouteTreatment(lineId uuid.UUID, userId uuid.UUID) (content []byte, err error) {
	data := requests.TreatmentRequest{
		LineID: lineId,
		UserId: userId,
	}

	jsonData, err := json.Marshal(data)

	return invoke("POST", "/line/appoint/start/", "application/json", jsonData)
}

func RerouteTreatmentToSpec(lineId uuid.UUID, userId uuid.UUID, specId uuid.UUID) (content []byte, err error) {
	data := requests.TreatmentWithSpecRequest{
		LineID: lineId,
		UserId: userId,
		SpecId: specId,
	}

	jsonData, err := json.Marshal(data)

	return invoke("POST", "/line/appoint/spec/", "application/json", jsonData)
}

func invoke(method string, methodUrl string, contentType string, body []byte) (content []byte, err error) {
	methodUrl = strings.Trim(methodUrl, "/")
	reqUrl := cnf.Connect.Server + "/v1/" + methodUrl + "/"

	req, err := http.NewRequest(method, reqUrl, bytes.NewBuffer(body))
	if err != nil {
		logger.Warning("Error while create request for", reqUrl, "with method", method, ":", err)
	}

	req.SetBasicAuth(cnf.Connect.Login, cnf.Connect.Password)
	req.Header.Set("Content-Type", contentType)

	logger.Debug("---> request", req.Method, reqUrl)

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		logger.Debug("<--- request", req.Method, reqUrl, "with body", bodyBytes)
		if err != nil {
			logger.Warning("Error while read response body", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, &HttpError{
				Url:     req.URL.String(),
				Code:    resp.StatusCode,
				Message: string(bodyBytes),
			}
		}

		return bodyBytes, nil
	}
}
