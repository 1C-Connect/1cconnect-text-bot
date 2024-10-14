package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connect-text-bot/bot/requests"
	"connect-text-bot/internal/config"
	"connect-text-bot/internal/logger"

	"github.com/google/uuid"
)

var (
	client = &http.Client{
		Timeout: 25 * time.Second,
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

func SetHook(cnf *config.Conf, lineId uuid.UUID) (content []byte, err error) {
	data := requests.HookSetupRequest{
		Id:   lineId,
		Type: "bot",
		Url:  cnf.Server.Host + "/connect-push/receive/",
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return Invoke(cnf, http.MethodPost, "/hook/", nil, "application/json", jsonData)
}

func DeleteHook(cnf *config.Conf, lineId uuid.UUID) (content []byte, err error) {
	return Invoke(cnf, http.MethodDelete, "/hook/bot/"+lineId.String()+"/", nil, "application/json", nil)
}

func Invoke(cnf *config.Conf, method string, methodUrl string, url_params url.Values, contentType string, body []byte) (content []byte, err error) {
	methodUrl = strings.Trim(methodUrl, "/")
	reqUrl := cnf.Connect.Server + "/v1/" + methodUrl + "/"
	if url_params != nil {
		reqUrl += "?" + url_params.Encode()
	}

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
