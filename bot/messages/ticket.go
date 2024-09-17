package messages

import (
	"connect-text-bot/bot/client"
	"connect-text-bot/bot/requests"
	"connect-text-bot/config"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Получение заявки Service Desk по ID
func (msg *Message) GetTicket(c *gin.Context, id uuid.UUID) (content requests.Ticket, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)

	r, err := client.Invoke(cnf, http.MethodGet, "/ticket/"+id.String()+"/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение данных для заявок
func (msg *Message) GetTicketData(c *gin.Context) (content []requests.GetTicketDataResponse, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)

	r, err := client.Invoke(cnf, http.MethodGet, "/ticket/data/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение видов услуг
func (msg *Message) GetTicketDataKinds(c *gin.Context, ticketData []requests.GetTicketDataResponse) (kinds []requests.TicketDataKind, err error) {

	if len(ticketData) == 0 {
		ticketData, err = msg.GetTicketData(c)
		if err != nil {
			return
		}
	}

	for _, v := range ticketData {
		for _, value := range v.Kinds {
			for _, line := range value.Lines {
				if line == msg.LineId {
					kinds = append(kinds, value)
				}
			}
		}
	}

	return
}

// Получение всех типов услуг
func (msg *Message) GetTicketDataAllTypes(c *gin.Context, ticketData []requests.GetTicketDataResponse) (types []requests.TicketDataType, err error) {

	if len(ticketData) == 0 {
		ticketData, err = msg.GetTicketData(c)
		if err != nil {
			return
		}
	}

	for _, v := range ticketData {
		types = append(types, v.Types...)
	}

	return
}

// Получение типов услуг у определенной услуги
func (msg *Message) GetTicketDataTypesWhereKind(c *gin.Context, ticketData []requests.GetTicketDataResponse, kindId uuid.UUID) (types []requests.TicketDataType, err error) {

	if len(ticketData) == 0 {
		ticketData, err = msg.GetTicketData(c)
		if err != nil {
			return
		}
	}

	allKinds, err := msg.GetTicketDataKinds(c, ticketData)
	if err != nil {
		return
	}

	allTypes, err := msg.GetTicketDataAllTypes(c, ticketData)
	if err != nil {
		return
	}

	for _, kind := range allKinds {
		if kind.ID == kindId {
			for _, kindType := range kind.Types {
				for _, type_ := range allTypes {
					if type_.ID == kindType {
						types = append(types, type_)
					}
				}
			}
		}
	}

	return
}
