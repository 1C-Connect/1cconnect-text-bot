package requests

import "github.com/google/uuid"

type (
	TreatmentRequest struct {
		LineID uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserID uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	}

	TreatmentWithSpecRequest struct {
		LineID uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserID uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		SpecID uuid.UUID `json:"spec_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	}

	TreatmentWithSpecAndAuthorRequest struct {
		LineID   uuid.UUID  `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserID   uuid.UUID  `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		SpecID   uuid.UUID  `json:"spec_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		AuthorID *uuid.UUID `json:"author_id,omitempty" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	}

	TreatmentReroute struct {
		LineID   uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserID   uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		ToLineID uuid.UUID `json:"to_line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		Quote    string    `json:"quote,omitempty" example:"цитата"`
	}
)
