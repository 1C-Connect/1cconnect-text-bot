package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"connect-text-bot/internal/connect/requests"

	"github.com/google/uuid"
)

func (c Client) RerouteTreatment(ctx context.Context, userID uuid.UUID) error {
	data := requests.TreatmentRequest{
		LineID: c.lineID,
		UserID: userID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.Invoke(ctx, http.MethodPost, "/line/appoint/start/", nil, "application/json", jsonData)

	return err
}

// Закрыть текущее обращение
func (c Client) CloseTreatment(ctx context.Context, userID uuid.UUID) error {
	time.Sleep(500 * time.Millisecond)

	data := requests.TreatmentRequest{
		LineID: c.lineID,
		UserID: userID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.Invoke(ctx, http.MethodPost, "/line/drop/treatment/", nil, "application/json", jsonData)

	return err
}

func (c Client) StartAndReroute(ctx context.Context, userID uuid.UUID) error {
	_ = c.Start(ctx, userID)
	data := requests.TreatmentRequest{
		LineID: c.lineID,
		UserID: userID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.Invoke(ctx, http.MethodPost, "/line/appoint/start/", nil, "application/json", jsonData)

	return err
}

// Перевод обращения на другую линию
func (c Client) Reroute(ctx context.Context, userID, toLineID uuid.UUID, quote string) error {
	data := requests.TreatmentRerouteRequest{
		LineID:   c.lineID,
		UserID:   userID,
		ToLineID: toLineID,
		Quote:    quote,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.Invoke(ctx, http.MethodPost, "/line/reroute/", nil, "application/json", jsonData)

	return err
}

// Попытаться назначить конкретного специалиста
func (c Client) AppointSpec(ctx context.Context, userID uuid.UUID, authorID *uuid.UUID, appointSpec uuid.UUID) error {

	data := requests.TreatmentWithSpecAndAuthorRequest{
		LineID:   c.lineID,
		UserID:   userID,
		SpecID:   appointSpec,
		AuthorID: authorID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.Invoke(ctx, http.MethodPost, "/line/appoint/spec/", nil, "application/json", jsonData)

	return err
}
