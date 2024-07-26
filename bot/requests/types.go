package requests

import (
	"time"

	"github.com/google/uuid"
)

type (
	// Описание объекта - Пользователи
	Users []User
	// Описание объекта - Пользователь
	User struct {
		UserId             uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		CounterpartId      uuid.UUID `json:"counterpart_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		CounterpartOwnerId uuid.UUID `json:"counterpart_owner_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		Name               string    `json:"name"`
		Surname            string    `json:"surname"`
		Patronymic         string    `json:"patronymic"`
		AvatarUrl          string    `json:"avatar_url,omitempty"`
		AvatarSmallUrl     string    `json:"avatar_small_url,omitempty"`
		Email              string    `json:"email"`
		Post               string    `json:"post"`
		Phone              string    `json:"phone"`
	}

	// Описание объекта - Компетенции
	Competences []Competence
	// Описание объекта - Компетенция
	Competence struct {
		LineId       uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		SpecialistId uuid.UUID `json:"specialist_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		PoolPriority int8      `json:"pool_priority"`
		IsFranchSpec bool      `json:"is_franch_spec"`
	}

	// Описание объекта - Линии, подключенные пользователям
	Subscriptions []Subscription
	// Описание объекта - Линия, подключенная пользователю
	Subscription struct {
		LineId uuid.UUID `json:"line_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		UserId uuid.UUID `json:"user_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		// Дата начала подписки
		SubscriptionSet time.Time `json:"subscription_set"`
		// Дата окончания подписки. nil когда бессрочная подписка
		SubscriptionExpireAt time.Time `json:"subscription_expire_at,omitempty"`
	}
)
