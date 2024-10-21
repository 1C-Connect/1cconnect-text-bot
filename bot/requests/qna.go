package requests

import "github.com/google/uuid"

type (
	Qna struct {
		LineID        uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserID        uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		SkipGreetings bool      `json:"skip_greetings" binding:"omitempty" example:"false"`
		SkipGoodbyes  bool      `json:"skip_goodbyes" binding:"omitempty" example:"false"`
	}
)
