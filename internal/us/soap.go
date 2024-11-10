package us

import (
	"context"

	"connect-text-bot/internal/database"

	"github.com/google/uuid"
	"github.com/hooklift/gowsdl/soap"
)

// Создание заявки на обслуживание
func CreateTicket(ctx context.Context, soapcl *soap.Client, userID, lineID uuid.UUID, ticket database.Ticket) (content map[string]string, err error) {
	service := NewPartnerWebAPI2PortType(soapcl)

	serviceRequestAdd, err := service.ServiceRequestAddContext(
		ctx,
		&ServiceRequestAdd{
			Xs: "http://www.w3.org/2001/XMLSchema",
			Params: &Params{
				Property: []ParamsProperty{
					formProperty("ServiceRequestChannelID", XsString, ticket.ChannelID.String()),
					formProperty("ServiceLineKindID", XsString, lineID.String()),
					formProperty("ServiceKindID", XsString, ticket.Service.ID.String()),
					formProperty("ServiceRequestTypeID", XsString, ticket.ServiceType.ID.String()),
					formProperty("UserID", XsString, userID.String()),
					formProperty("ExecutorID", XsString, ticket.Executor.ID.String()),
					formProperty("Description", XsString, ticket.Description),
					formProperty("Summary", XsString, ticket.Theme),
				},
			},
		},
	)
	if err != nil {
		return
	}

	r, err := serviceRequestAdd.Return_.GetResult()
	if err != nil {
		return
	}

	content = make(map[string]string)
	for _, v := range r {
		content[v.Name] = v.Value.Text
	}

	return
}

// сформировать поле для запроса
func formProperty(name, valueType, valueText string) ParamsProperty {
	return ParamsProperty{
		Name: name,
		Value: PropertyValue{
			Type: valueType,
			Text: valueText,
		},
	}
}
