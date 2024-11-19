package cache

import (
	"connect-text-bot/internal/botconfig_parser"
	"connect-text-bot/internal/connect/response"
	"connect-text-bot/internal/database"
)

type (
	// набор данных привязываемые к пользователю бота
	Chat struct {
		// история состояний от start до где щас пользователь
		HistoryState []string `json:"history_state"`
		// предыдущее состояние
		PreviousState string `json:"prev_state" binding:"required" example:"100"`
		// текущее состояние
		CurrentState string `json:"curr_state" binding:"required" example:"300"`
		// информация о пользователе
		User response.User `json:"user"`

		// хранимые данные
		Vars map[string]string `json:"vars" binding:"omitempty"`
		// хранимые данные о заявке
		Ticket database.Ticket `json:"ticket" binding:"omitempty"`
		// кнопка которую необходимо сохранить для последующей работы
		SavedButton *botconfig_parser.Button `json:"saved_button" binding:"omitempty"`
	}
)
