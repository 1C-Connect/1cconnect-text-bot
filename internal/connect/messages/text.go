package messages

import (
	"github.com/google/uuid"
)

type MessageType int

const (
	MESSAGE_TEXT                      MessageType = 1
	MESSAGE_CALL_START_TREATMENT      MessageType = 20
	MESSAGE_CALL_START_NO_TREATMENT   MessageType = 21
	MESSAGE_FILE                      MessageType = 70
	MESSAGE_TREATMENT_START_BY_USER   MessageType = 80
	MESSAGE_TREATMENT_START_BY_SPEC   MessageType = 81
	MESSAGE_TREATMENT_CLOSE           MessageType = 82
	MESSAGE_NO_FREE_SPECIALISTS       MessageType = 83
	MESSAGE_LINE_REROUTING_OTHER_LINE MessageType = 89
	MESSAGE_TREATMENT_CLOSE_ACTIVE    MessageType = 90
	MESSAGE_TREATMENT_TO_BOT          MessageType = 200
)

type (
	Message struct {
		LineID uuid.UUID `json:"line_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		UserID uuid.UUID `json:"user_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`

		MessageID     uuid.UUID   `json:"message_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		MessageType   MessageType `json:"message_type" binding:"required" example:"1"`
		MessageAuthor *uuid.UUID  `json:"author_id" binding:"omitempty" example:"4e48509f-6366-4897-9544-46f006e47074"`
		MessageTime   string      `json:"message_time" binding:"required" example:"1"`
		Text          string      `json:"text" example:"Привет"`
		Data          struct {
			Redirect string `json:"redirect"`
		} `json:"data"`
	}

	AutofaqAnswer struct {
		ID           uuid.UUID `json:"id"`
		Text         string    `json:"text"`
		Accuracy     float32   `json:"accuracy"`
		AnswerSource string    `json:"answer_source"`
	}

	AutofaqRequestBody struct {
		RequestID uuid.UUID       `json:"request_id"`
		Question  string          `json:"question"`
		Answers   []AutofaqAnswer `json:"answers"`
	}
)
