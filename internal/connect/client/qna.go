package client

import (
	"context"
	"encoding/json"
	"net/http"

	"connect-text-bot/internal/connect/messages"
	"connect-text-bot/internal/logger"

	"github.com/google/uuid"
)

// GetQNA - Метод позволяет получить варианты ответов на вопрос пользователя в сервисе AutoFAQ.
func (c Client) GetQNA(ctx context.Context, userID uuid.UUID, skipGreetings, skipGoodbyes bool) (resp *messages.AutofaqRequestBody, err error) {
	data := messages.Qna{
		LineID:        c.lineID,
		UserID:        userID,
		SkipGreetings: skipGreetings,
		SkipGoodbyes:  skipGoodbyes,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Warning("text - GetQNA", err)
		return
	}

	body, err := c.Invoke(ctx, http.MethodPost, "/line/qna/", nil, "application/json", jsonData)
	if err != nil {
		logger.Warning("text - GetQNA", err)
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		logger.Warning("text - GetQNA", err)
		return
	}

	// Debug
	logger.Debug("text - GetQNA", resp)

	return
}

// Отметить выбранный вариант подсказки
func (c Client) QnaSelected(ctx context.Context, requestID, resultID uuid.UUID) {
	data := messages.Selected{
		RequestID: requestID,
		ResultID:  resultID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Warning("text - GetQNA", err)
		return
	}

	body, err := c.Invoke(ctx, http.MethodPut, "/line/qna/selected/", nil, "application/json", jsonData)
	if err != nil {
		logger.Warning("text - QnaSelected", err, body)
	}
}
