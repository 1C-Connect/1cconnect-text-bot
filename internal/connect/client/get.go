package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"

	"connect-text-bot/internal/connect/response"

	"github.com/google/uuid"
)

// Проверить доступен ли специалист по id
func (c Client) GetSpecialistAvailable(ctx context.Context, specID uuid.UUID) (available bool, err error) {
	specIDs, err := c.GetSpecialistsAvailable(ctx)
	if err != nil {
		return
	}

	return slices.Contains(specIDs, specID), err
}

// Получить список специалистов доступных по линии
func (c Client) GetSpecialistsAvailable(ctx context.Context) (specIDs []uuid.UUID, err error) {
	r, err := c.Invoke(ctx, http.MethodGet, "/line/specialists/"+c.lineID.String()+"/available/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &specIDs)
	return
}

// Получение информации о пользователе
func (c Client) GetSubscriber(ctx context.Context, userID uuid.UUID) (content response.User, err error) {
	r, err := c.Invoke(ctx, http.MethodGet, "/line/subscriber/"+userID.String()+"/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение списка линий, подключенных пользователям
func (c Client) GetSubscriptions(ctx context.Context, userID, lineID uuid.UUID) (content response.Subscriptions, err error) {
	var v = url.Values{}
	v.Add("user_id", userID.String())
	v.Add("line_id", lineID.String())

	r, err := c.Invoke(ctx, http.MethodGet, "/line/subscriptions/", v, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение информации о специалисте
func (c Client) GetSpecialist(ctx context.Context, specID uuid.UUID) (content response.User, err error) {
	r, err := c.Invoke(ctx, http.MethodGet, "/line/specialist/"+specID.String()+"/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение информации о специалистах
func (c Client) GetSpecialists(ctx context.Context, lineID uuid.UUID) (content response.Users, err error) {
	var v = url.Values{}
	if lineID != uuid.Nil {
		v.Add("line_id", lineID.String())
	}

	r, err := c.Invoke(ctx, http.MethodGet, "/line/specialists/", v, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}
