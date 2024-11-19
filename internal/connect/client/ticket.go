package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"connect-text-bot/internal/connect/response"

	"github.com/google/uuid"
)

// Получение заявки Service Desk по ID
func (c Client) GetTicket(ctx context.Context, id uuid.UUID) (content response.Ticket, err error) {
	r, err := c.Invoke(ctx, http.MethodGet, "/ticket/"+id.String()+"/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение данных для заявок
func (c Client) GetTicketData(ctx context.Context, userCounterpartOwnerID uuid.UUID) (result response.GetTicketDataResponse, err error) {
	v := url.Values{}
	v.Add("line_id", c.lineID.String())

	r, err := c.Invoke(ctx, http.MethodGet, "/ticket/data/", v, "application/json", nil)
	if err != nil {
		return
	}

	var content []response.GetTicketDataResponse
	err = json.Unmarshal(r, &content)
	if err != nil {
		return
	}

	for _, v := range content {
		if v.CounterpartID == userCounterpartOwnerID {
			result = v
			break
		}
	}

	return
}

// Получение видов услуг
func (c Client) GetTicketDataKinds(ctx context.Context, ticketData *response.GetTicketDataResponse, userCounterpartOwnerID uuid.UUID) (kinds []response.TicketDataKind, err error) {
	if ticketData == nil {
		ticketData = new(response.GetTicketDataResponse)
		*ticketData, err = c.GetTicketData(ctx, userCounterpartOwnerID)
		if err != nil {
			return
		}
	}

	// получаем все виды услуг доступные по текущей линии
	for _, value := range ticketData.Kinds {
		for _, line := range value.Lines {
			if line == c.lineID {
				kinds = append(kinds, value)
				break
			}
		}
	}

	return
}

// Получение всех типов услуг
func (c Client) GetTicketDataAllTypes(ctx context.Context, ticketData *response.GetTicketDataResponse, userCounterpartOwnerID uuid.UUID) (types []response.TicketDataType, err error) {
	if ticketData == nil {
		ticketData = new(response.GetTicketDataResponse)
		*ticketData, err = c.GetTicketData(ctx, userCounterpartOwnerID)
		if err != nil {
			return
		}
	}

	types = ticketData.Types

	return
}

// Получение типов услуг у определенной услуги
func (c Client) GetTicketDataTypesWhereKind(ctx context.Context, ticketData *response.GetTicketDataResponse, userCounterpartOwnerID uuid.UUID, kindID uuid.UUID) (types []response.TicketDataType, err error) {
	if ticketData == nil {
		ticketData = new(response.GetTicketDataResponse)
		*ticketData, err = c.GetTicketData(ctx, userCounterpartOwnerID)
		if err != nil {
			return
		}
	}

	allKinds, err := c.GetTicketDataKinds(ctx, ticketData, userCounterpartOwnerID)
	if err != nil {
		return
	}

	allTypes, err := c.GetTicketDataAllTypes(ctx, ticketData, userCounterpartOwnerID)
	if err != nil {
		return
	}

	// ищем среди всех услуг виды работ которые доступны по выбранной услуге
	for _, kind := range allKinds {
		if kind.ID == kindID {
			for _, kindType := range kind.Types {
				for _, type_ := range allTypes {
					if type_.ID == kindType {
						types = append(types, type_)
					}
				}
			}
			break
		}
	}

	return
}
