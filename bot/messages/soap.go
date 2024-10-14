package messages

import (
	"connect-text-bot/internal/config"
	"connect-text-bot/internal/database"
	"connect-text-bot/internal/us"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hooklift/gowsdl/soap"
)

func createSoapClient(cnf *config.Conf) *soap.Client {
	return soap.NewClient(
		cnf.Connect.SoapServer,
		soap.WithBasicAuth(cnf.Connect.Login, cnf.Connect.Password),
	)
}

// Создание заявки на обслуживание
func (msg *Message) ServiceRequestAdd(c *gin.Context, ticket database.Ticket) (content map[string]string, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)

	soapClient := createSoapClient(cnf)

	service := us.NewPartnerWebAPI2PortType(soapClient)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	serviceRequestAdd, err := service.ServiceRequestAddContext(
		ctx,
		&us.ServiceRequestAdd{
			Xs: "http://www.w3.org/2001/XMLSchema",
			Params: &us.Params{
				Property: []us.ParamsProperty{
					formProperty("ServiceRequestChannelID", us.XsString, ticket.ChannelID.String()),
					formProperty("ServiceLineKindID", us.XsString, msg.LineId.String()),
					formProperty("ServiceKindID", us.XsString, ticket.Service.ID.String()),
					formProperty("ServiceRequestTypeID", us.XsString, ticket.ServiceType.ID.String()),
					formProperty("UserID", us.XsString, msg.UserId.String()),
					formProperty("ExecutorID", us.XsString, ticket.Executor.ID.String()),
					formProperty("Description", us.XsString, ticket.Description),
					formProperty("Summary", us.XsString, ticket.Theme),
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
func formProperty(name, valueType, valueText string) us.ParamsProperty {
	return us.ParamsProperty{
		Name: name,
		Value: us.PropertyValue{
			Type: valueType,
			Text: valueText,
		},
	}
}
