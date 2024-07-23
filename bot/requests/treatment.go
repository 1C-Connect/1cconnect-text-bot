package requests

import "github.com/google/uuid"

type (
	TreatmentRequest struct {
		LineID uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserId uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	}

	TreatmentWithSpecRequest struct {
		LineID uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserId uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		SpecId uuid.UUID `json:"spec_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	}

	TreatmentWithSpecAndAuthorRequest struct {
		LineID   uuid.UUID  `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserId   uuid.UUID  `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		SpecId   uuid.UUID  `json:"spec_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		AuthorID *uuid.UUID `json:"author_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	}
)
