package requests

import "github.com/google/uuid"

type (
	Selected struct {
		RequestID uuid.UUID `json:"request_id"  example:"eae98975-013f-4593-aba8-bf5b749b977d"`
		ResultID  uuid.UUID `json:"result_id"  example:"037bf6b3-a566-4365-af37-5bb97adf06a8"`
	}
)
