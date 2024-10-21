package database

import "github.com/google/uuid"

const (
	GREETINGS = "greetings"
	START     = "start"
	FINAL     = "final_menu"
	FAIL_QNA  = "fail_qna_menu"
	// ожидание сообщения пользователя
	WAIT_SEND = "wait_send_menu"
	// регистрация заявки
	CREATE_TICKET            = "create_ticket"
	CREATE_TICKET_PREV_STAGE = "create_ticket_prev_stage"
)

const (
	// переменная в Vars в которой хранится имя переменной которую надо редактировать следующим шагом
	VAR_FOR_SAVE = "VAR_FOR_SAVE"
)

// данные для формирования заявки
type (
	Ticket struct {
		ChannelID   uuid.UUID
		Theme       string
		Description string
		Executor    TicketPart
		Service     TicketPart
		ServiceType TicketPart
	}

	TicketPart struct {
		ID   uuid.UUID
		Name *string
	}
)

func (_ *Ticket) GetChannel() string     { return "channel" }
func (_ *Ticket) GetTheme() string       { return "theme" }
func (_ *Ticket) GetDescription() string { return "description" }
func (_ *Ticket) GetExecutor() string    { return "executor" }
func (_ *Ticket) GetService() string     { return "service" }
func (_ *Ticket) GetServiceType() string { return "type" }
