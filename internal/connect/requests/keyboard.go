package requests

import "github.com/google/uuid"

type (
	DropKeyboardRequest struct {
		LineID uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserID uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	}
)
