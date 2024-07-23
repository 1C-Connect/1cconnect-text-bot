package requests

import "github.com/google/uuid"

// Описание объекта - Пользователь
type User struct {
	UserId             uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	CounterpartId      uuid.UUID `json:"counterpart_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	CounterpartOwnerId uuid.UUID `json:"counterpart_owner_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	Name               string    `json:"name"`
	Surname            string    `json:"surname"`
	Patronymic         string    `json:"patronymic"`
	AvatarUrl          *string   `json:"avatar_url"`
	AvatarSmallUrl     *string   `json:"avatar_small_url"`
	Email              string    `json:"email"`
	Post               string    `json:"post"`
	Phone              string    `json:"phone"`
}

// Описание объекта - Компетенция
type Competence struct {
	LineId       uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	SpecialistId uuid.UUID `json:"specialist_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
	PoolPriority int8      `json:"pool_priority"`
	IsFranchSpec bool      `json:"is_franch_spec"`
}
