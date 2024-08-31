package database

type (
	Chat struct {
		PreviousState string            `json:"prev_state" binding:"required" example:"100"`
		CurrentState  string            `json:"curr_state" binding:"required" example:"300"`
		Vars          map[string]string `json:"vars" binding:"omitempty"`
	}
)

const (
	GREETINGS = "greetings"
	START     = "start"
	FINAL     = "final_menu"
	FAIL_QNA  = "fail_qna_menu"
	// ожидание сообщения пользователя
	WAIT_SEND = "wait_send"
)

const (
	// переменная в Vars в которой хранится имя переменной которую надо редактировать следующим шагом
	VAR_FOR_SAVE = "VAR_FOR_SAVE"
	// переменная в Vars для обозначения куда выполнить переход после завершения шага
	VAR_FOR_GOTO = "VAR_FOR_GOTO"
)
