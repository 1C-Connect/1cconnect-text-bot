package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connect-text-bot/internal/connect/requests"
	"connect-text-bot/internal/logger"

	"github.com/google/uuid"
)

type (
	Client struct {
		lineID uuid.UUID

		serverAddr string
		login      string
		password   string

		generalSettings bool

		specID *uuid.UUID

		cl *http.Client
	}

	HttpError struct {
		Url     string
		Code    int
		Message string
	}
)

func New(lineID uuid.UUID, server_addr, login, password string, generalSettings bool, specID *uuid.UUID) *Client {
	return &Client{
		lineID: lineID,

		serverAddr: server_addr,
		login:      login,
		password:   password,

		generalSettings: generalSettings,

		specID: specID,

		cl: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				IdleConnTimeout:     30 * time.Second,
				DisableKeepAlives:   false,
				MaxIdleConnsPerHost: 5,
				DisableCompression:  true,
			},
		},
	}
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("Http request failed for %s with code %d and message:\n%s", e.Url, e.Code, e.Message)
}

func (c *Client) SetHook(hookAddr string) (content []byte, err error) {
	data := requests.HookSetupRequest{
		ID:   c.lineID,
		Type: "bot",
		Url:  hookAddr,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return c.Invoke(context.Background(), http.MethodPost, "/hook/", nil, "application/json", jsonData)
}

func (c *Client) DeleteHook() (content []byte, err error) {
	return c.Invoke(context.Background(), http.MethodDelete, "/hook/bot/"+c.lineID.String()+"/", nil, "application/json", nil)
}

func (c *Client) Invoke(ctx context.Context, method string, methodUrl string, urlParams url.Values, contentType string, body []byte) (content []byte, err error) {
	methodUrl = strings.Trim(methodUrl, "/")
	reqUrl := c.serverAddr + "/v1/" + methodUrl + "/"
	if urlParams != nil {
		reqUrl += "?" + urlParams.Encode()
	}

	req, err := http.NewRequest(method, reqUrl, bytes.NewBuffer(body))
	if err != nil {
		logger.Warning("Error while create request for", reqUrl, "with method", method, ":", err)
	}

	req.SetBasicAuth(c.login, c.password)
	req.Header.Set("Content-Type", contentType)

	logger.Debug("---> request", req.Method, reqUrl)

	resp, err := c.cl.Do(req)

	if err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
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
