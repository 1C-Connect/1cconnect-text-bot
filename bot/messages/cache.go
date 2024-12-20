package messages

import (
	"connect-text-bot/bot/requests"
	"connect-text-bot/internal/botconfig_parser"
	"connect-text-bot/internal/database"
	"connect-text-bot/internal/logger"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/allegro/bigcache/v3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		User requests.User `json:"user"`

		// хранимые данные
		Vars map[string]string `json:"vars" binding:"omitempty"`
		// хранимые данные о заявке
		Ticket database.Ticket `json:"ticket" binding:"omitempty"`
		// кнопка которую необходимо сохранить для последующей работы
		SavedButton *botconfig_parser.Button `json:"saved_button" binding:"omitempty"`
	}
)

func (msg *Message) ChangeCache(c *gin.Context, chatState *Chat) error {
	cache := c.MustGet("cache").(*bigcache.BigCache)

	data, err := json.Marshal(chatState)
	if err != nil {
		logger.Warning("Error while change state to cache", err)
		return err
	}

	dbStateKey := msg.UserID.String() + ":" + msg.LineID.String()

	err = cache.Set(dbStateKey, data)
	logger.Debug("Write state to cache result")
	if err != nil {
		logger.Warning("Error while write state to cache", err)
	}

	return nil
}

func (msg *Message) ChangeCacheTicket(c *gin.Context, chatState *Chat, key string, value database.TicketPart) error {
	t := database.Ticket{}

	switch key {
	case t.GetChannel():
		if value.ID == uuid.Nil {
			return fmt.Errorf("не передан ID для: %s", key)
		}
		chatState.Ticket.ChannelID = value.ID
	case t.GetTheme(), t.GetDescription():
		if value.Name == nil {
			return fmt.Errorf("не передан Name для: %s", key)
		}
		switch key {
		case t.GetTheme():
			chatState.Ticket.Theme = *value.Name
		case t.GetDescription():
			chatState.Ticket.Description = *value.Name
		}
	case t.GetExecutor(), t.GetService(), t.GetServiceType():
		if value.ID == uuid.Nil || value.Name == nil {
			return fmt.Errorf("не передан ID и/или Name для: %s", key)
		}
		switch key {
		case t.GetExecutor():
			chatState.Ticket.Executor = value
		case t.GetService():
			chatState.Ticket.Service = value
		case t.GetServiceType():
			chatState.Ticket.ServiceType = value
		}
	}

	return msg.ChangeCache(c, chatState)
}

func (msg *Message) ChangeCacheVars(c *gin.Context, chatState *Chat, key, value string) error {
	if chatState.Vars == nil {
		chatState.Vars = make(map[string]string)
	}
	chatState.Vars[key] = value

	return msg.ChangeCache(c, chatState)
}

func (msg *Message) ChangeCacheSavedButton(c *gin.Context, chatState *Chat, button *botconfig_parser.Button) error {
	chatState.SavedButton = button

	return msg.ChangeCache(c, chatState)
}

func (msg *Message) ChangeCacheState(c *gin.Context, chatState *Chat, toState string) error {
	if chatState.CurrentState == toState {
		return nil
	}

	chatState.PreviousState = chatState.CurrentState
	chatState.CurrentState = toState
	chatState.HistoryStateAppend(toState)

	return msg.ChangeCache(c, chatState)
}

func (msg *Message) GetState(c *gin.Context) Chat {
	cache := c.MustGet("cache").(*bigcache.BigCache)

	var chatState Chat

	dbStateKey := msg.UserID.String() + ":" + msg.LineID.String()

	b, err := cache.Get(dbStateKey)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			logger.Info("No state in cache for " + msg.UserID.String() + ":" + msg.LineID.String())
			chatState = Chat{
				PreviousState: database.GREETINGS,
				CurrentState:  database.GREETINGS,
			}

			// сохраняем пользовательские данные
			err := msg.SaveUserDataInCache(c, &chatState)
			if err != nil {
				logger.Warning("Error while get user data", err)
			}
			return chatState
		}
	}
	err = json.Unmarshal(b, &chatState)
	if err != nil {
		logger.Warning("Error while decoding state", err)
	}

	return chatState
}

// получить значение переменной из хранимых данных
func (msg *Message) GetCacheUserInfo(c *gin.Context) requests.User {
	state := msg.GetState(c)

	return state.User
}

// получить значение переменной из хранимых данных
func (msg *Message) GetCacheVar(c *gin.Context, varName string) (string, bool) {
	state := msg.GetState(c)

	result, exist := state.Vars[varName]
	return result, exist
}

// чистим необязательные поля хранимых данных
func (msg *Message) ClearCacheOmitemptyFields(c *gin.Context, chatState *Chat) error {
	if _, exist := chatState.Vars[database.VAR_FOR_SAVE]; exist {
		chatState.Vars[database.VAR_FOR_SAVE] = ""
	}
	chatState.SavedButton = nil
	chatState.Ticket = database.Ticket{}

	return msg.ChangeCache(c, chatState)
}

// сохранить данные о пользователе в кеше
func (msg *Message) SaveUserDataInCache(c *gin.Context, chatState *Chat) (err error) {
	// получаем данные о пользователе
	userData, err := msg.GetSubscriber(c)
	if err != nil {
		return
	}

	chatState.User = userData
	return
}

// вернуться на предыдущий пункт меню в истории
func (chatState *Chat) HistoryStateBack() {
	// если истории нет то мы в старт должны быть
	if len(chatState.HistoryState) == 0 {
		chatState.PreviousState = database.GREETINGS
		return
	}

	// удаление последнего меню в истории
	lastIndex := len(chatState.HistoryState) - 1
	chatState.HistoryState = chatState.HistoryState[:lastIndex]

	// если после удаления последнего меню не осталось истории
	if len(chatState.HistoryState) == 0 {
		chatState.PreviousState = database.GREETINGS
		return
	}
	chatState.PreviousState = chatState.HistoryState[lastIndex-1]
}

// добавить новый пункт меню в историю
func (chatState *Chat) HistoryStateAppend(state string) {
	// чистим историю если меню последнее должно быть
	if slices.Contains([]string{database.FAIL_QNA, database.FINAL, database.START, database.GREETINGS}, state) {
		chatState.HistoryStateClear()
		return
	}

	// игнорируем добавление если спец кнопка
	if slices.Contains([]string{database.CREATE_TICKET, database.CREATE_TICKET_PREV_STAGE, database.WAIT_SEND}, state) {
		return
	}

	// если история пустая или последний элемент совпадает с переданным
	if len(chatState.HistoryState) == 0 || chatState.HistoryState[len(chatState.HistoryState)-1] != state {
		chatState.HistoryState = append(chatState.HistoryState, state)
	}
}

// очистить историю
func (chatState *Chat) HistoryStateClear() {
	chatState.HistoryState = []string{}
}
