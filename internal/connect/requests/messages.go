package requests

import "github.com/google/uuid"

type (
	KeyboardKey struct {
		ID   string `json:"id" example:"123"`
		Text string `json:"text" example:"Расскажи анекдот"`
	}

	MessageRequest struct {
		LineID   uuid.UUID        `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserID   uuid.UUID        `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		AuthorID *uuid.UUID       `json:"author_id,omitempty" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		Text     string           `json:"text" example:"Hello world!"`
		Keyboard *[][]KeyboardKey `json:"keyboard"`
	}

	FileRequest struct {
		LineID   uuid.UUID        `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserID   uuid.UUID        `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		AuthorID *uuid.UUID       `json:"author_id,omitempty" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		FileName string           `json:"file_name" example:"text.pdf"`
		Comment  *string          `json:"comment" binding:"omitempty" example:"Держи краба!"`
		Keyboard *[][]KeyboardKey `json:"keyboard"`
	}
)
