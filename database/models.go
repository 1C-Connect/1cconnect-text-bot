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
type Ticket struct {
	ChannelID   uuid.UUID
	Theme       string
	Description string
	Executor    struct {
		Id   uuid.UUID
		Name string
	}
	Service struct {
		Id   uuid.UUID
		Name string
	}
	ServiceType struct {
		Id   uuid.UUID
		Name string
	}
}

func (t *Ticket) GetChannel() string     { return "channel" }
func (t *Ticket) GetTheme() string       { return "theme" }
func (t *Ticket) GetDescription() string { return "description" }
func (t *Ticket) GetExecutor() string    { return "executor" }
func (t *Ticket) GetService() string     { return "service" }
func (t *Ticket) GetServiceType() string { return "type" }
