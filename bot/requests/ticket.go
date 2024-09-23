package requests

import (
	"time"

	"github.com/google/uuid"
)

type (
	GetTicketDataResponse struct {
		CounterpartID uuid.UUID           `json:"counterpart_id"`
		Name          string              `json:"name"`
		Channels      []TicketDataChannel `json:"channels"`
		Types         []TicketDataType    `json:"types"`
		Statuses      []TicketDataStatus  `json:"statuses"`
		Kinds         []TicketDataKind    `json:"kinds"`
	}

	TicketDataChannel struct {
		ID        uuid.UUID `json:"id" format:"uuid" example:"6bfc3cc7-03ee-4cf1-a652-857620c91197"`
		Name      string    `json:"name" example:"1С-Коннект"`
		Type      string    `json:"type" example:"CONNECT"`
		IsDefault bool      `json:"is_default" example:"true"`
	}

	TicketDataType struct {
		ID   uuid.UUID `json:"id" format:"uuid" example:"6bfc3cc7-03ee-4cf1-a652-857620c91197"`
		Name string    `json:"name" example:"Запрос на обслуживание"`

		AdditionalFields []TicketDataAdditionalField `json:"additional_fields"`
	}

	TicketDataAdditionalField struct {
		ID   string `json:"id" example:"FIELD1"`
		Name string `json:"name" example:"Поле 1"`
	}

	TicketDataStatus struct {
		ID   uuid.UUID `json:"id" format:"uuid" example:"6bfc3cc7-03ee-4cf1-a652-857620c91197"`
		Name string    `json:"name" example:"Выполняется"`
		Type string    `json:"type" example:"WORK"`

		StatusesTransitions []TicketDataStatusTransitions `json:"statuses_transitions"`
	}

	TicketDataStatusTransitions struct {
		NextStatusID    uuid.UUID `json:"next_status_id" format:"uuid" example:"49e43168-f886-4d52-bbe5-97b7d6f64ba2"`
		PermissionLevel string    `json:"permission_level" example:"EXECUTOR"`
	}

	TicketDataKind struct {
		ID          uuid.UUID   `json:"id" format:"uuid" example:"6bfc3cc7-03ee-4cf1-a652-857620c91197"`
		Name        string      `json:"name" example:"Запрос на обслуживание"`
		Description string      `json:"description" example:""`
		Types       []uuid.UUID `json:"types" example:"fb0facf1-c09e-4f9d-990f-58bb4bbe7af4,302c18b3-68be-4a06-b305-44c04afcb00e"`
		Lines       []uuid.UUID `json:"lines" example:"81ac7c26-fc65-44ba-a479-deb085ed9cfb,004a41a3-c2fb-4023-95d6-3721eeba6f71"`
	}
)

type (
	// Заявка Service Desk
	Ticket struct {
		// ID Заявки
		ID uuid.UUID `json:"id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		// ID Транзакции
		TransactionID uuid.UUID `json:"transaction_id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		// Номер заявки
		Number string `json:"number"`
		// Время создания
		CreatedAt time.Time `json:"created_at"`
		// Время изменения
		UpdatedAt time.Time `json:"updated_at"`
		// Описание
		Description string `json:"description,omitempty"`
		// Приоритет:
		// - LOW - низкий
		// - STANDARD - Стандартный
		// - HIGH - Высокий
		Priority string `json:"priority"`
		// Длительность работы, сек (0 - не задано)
		Duration int `json:"duration"`
		// Описание решения
		Result string `json:"result,omitempty"`
		// Тема
		Summary string `json:"summary,omitempty"`
		// Срок
		Deadline string `json:"deadline,omitempty"`
		// Вид работ
		Kind ServiceKind `json:"kind"`
		// Линия поддержки
		Line LineShort `json:"line"`
		// Тип заявки
		Type TicketType `json:"type"`
		// Канал связи
		Channel TicketChannel `json:"channel"`
		// Статус заявки
		Status TicketStatus `json:"status"`
		// Заказчик
		Initiator User `json:"initiator"`
		// Автор (отсутствует, если заявка создана в учетной системе)
		Author *User `json:"author,omitempty"`
		// Исполнитель
		Executor *User `json:"executor,omitempty"`
		// Значения дополнительных полей заявки
		Fields []TicketAdditionalFieldValue `json:"fields,omitempty"`
	}

	// Виды работ Service Desk
	ServiceKind struct {
		// ID Вида работ
		ID uuid.UUID `json:"id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		// Название
		Name string `json:"name"`
		// Описание
		Description string `json:"description,omitempty"`
	}

	// Линия поддержки (краткие сведения)
	LineShort struct {
		// ID линии поддержки
		ID uuid.UUID `json:"id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		// Название
		Name string `json:"name"`
	}

	// Типы заявок Service Desk
	TicketType struct {
		// ID Типа заявок
		ID uuid.UUID `json:"id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		// Название
		Name string `json:"name"`
	}

	// Каналы связи Service Desk
	TicketChannel struct {
		// ID Канала связи
		ID uuid.UUID `json:"id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		// Название
		Name string `json:"name"`
		// Тип:
		// - CONNECT - 1C-Коннект
		// - EMAIL - E-mail
		// - PHONE - Телефон
		// - OTHER - Прочее
		Type string `json:"type"`
	}

	// Статусы заявок Service Desk
	TicketStatus struct {
		// ID Статуса заявки
		ID uuid.UUID `json:"id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		// Название
		Name string `json:"name"`
		// Тип:
		// - NEW - Новая
		// - WORK - Выполняется
		// - VALIDATION - Валидация
		// - CONFIRMED - Подтверждена
		// - REJECTED - Отклонена
		// - SUSPENDED - Приостановлена
		// - FINISHED - Завершена
		// - CANCELLED - Отменена
		Type string `json:"type"`
	}

	// Дополнительное поле заявки Service Desk
	TicketAdditionalFieldValue struct {
		// Идентификатор поля:
		// - FIELD1 - Первое
		// - FIELD2 - Второе
		// - FIELD3 - Третье
		// - FIELD4 - Четвертое
		ID string `json:"id"`
		// Значение поля
		Value string `json:"value"`
	}
)
