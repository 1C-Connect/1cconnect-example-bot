package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"connect-companion/bot/requests"
	"connect-companion/config"
	"connect-companion/logger"

	"github.com/google/uuid"
)

var (
	cnf = &config.Conf{}
)

func Configure(c *config.Conf) {
	cnf = c
}

var (
	client = &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout:     30 * time.Second,
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 5,
			DisableCompression:  true,
		},
	}
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

func SetHook(lineId uuid.UUID) (content []byte, err error) {
	data := requests.HookSetupRequest{
		Id:   lineId,
		Type: "bot",
		Url:  cnf.Server.Host + "/connect-push/receive/",
		/*
			// Пример меню для перевода
			BotScenarioPoint: &[]requests.BotScenarioPoint{
				requests.BotScenarioPoint{
					Text: "Как добавить сотрудника в 1С-Коннект?",
					Data: "add_collegue,level:1",
				},
				requests.BotScenarioPoint{
					Text:        "Группа сценариев",
					Description: "Вложенный уровень",
					Data:        "add_collegue,level:1",
					Childs: &[]requests.BotScenarioPoint{
						requests.BotScenarioPoint{
							Text:        "Третий уровень заведения сотрудника",
							Description: "Если уже залогинен в УС",
							Data:        "add_collegue,level:3",
						},
					},
				},
			},
		*/
	}
	jsonData, err := json.Marshal(data)

	return Invoke("POST", "/hook/", "application/json", jsonData)
}

func DeleteHook(lineId uuid.UUID) (content []byte, err error) {
	return Invoke("DELETE", "/hook/bot/"+lineId.String()+"/", "application/json", nil)
}

func Invoke(method string, methodUrl string, contentType string, body []byte) (content []byte, err error) {
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
