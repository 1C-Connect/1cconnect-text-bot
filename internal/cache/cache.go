package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"connect-text-bot/internal/botconfig_parser"
	"connect-text-bot/internal/connect/client"
	"connect-text-bot/internal/database"
	"connect-text-bot/internal/logger"

	"github.com/allegro/bigcache/v3"
	"github.com/google/uuid"
)

func (chatState *Chat) ChangeCache(cache *bigcache.BigCache, userID, lineID uuid.UUID) error {
	data, err := json.Marshal(chatState)
	if err != nil {
		logger.Warning("Error while change state to cache", err)
		return err
	}

	dbStateKey := userID.String() + ":" + lineID.String()

	err = cache.Set(dbStateKey, data)
	logger.Debug("Write state to cache result")
	if err != nil {
		logger.Warning("Error while write state to cache", err)
	}

	return nil
}

func (chatState *Chat) ChangeCacheTicket(cache *bigcache.BigCache, userID, lineID uuid.UUID, key string, value database.TicketPart) error {
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
	default:
		return fmt.Errorf("не корректный ключ: %s", key)
	}

	return chatState.ChangeCache(cache, userID, lineID)
}

func (chatState *Chat) ChangeCacheVars(cache *bigcache.BigCache, userID, lineID uuid.UUID, key, value string) error {
	if chatState.Vars == nil {
		chatState.Vars = make(map[string]string)
	}
	chatState.Vars[key] = value

	return chatState.ChangeCache(cache, userID, lineID)
}

func (chatState *Chat) ChangeCacheSavedButton(cache *bigcache.BigCache, userID, lineID uuid.UUID, button *botconfig_parser.Button) error {
	chatState.SavedButton = button

	return chatState.ChangeCache(cache, userID, lineID)
}

func (chatState *Chat) ChangeCacheState(cache *bigcache.BigCache, userID, lineID uuid.UUID, toState string) error {
	if chatState.CurrentState == toState {
		return nil
	}

	chatState.PreviousState = chatState.CurrentState
	chatState.CurrentState = toState

	err := chatState.HistoryStateAppend(cache, userID, lineID, toState)
	if err != nil {
		return err
	}

	return chatState.ChangeCache(cache, userID, lineID)
}

// чистим необязательные поля хранимых данных
func (chatState *Chat) ClearCacheOmitemptyFields(cache *bigcache.BigCache, userID, lineID uuid.UUID) error {
	if _, exist := chatState.Vars[database.VAR_FOR_SAVE]; exist {
		chatState.Vars[database.VAR_FOR_SAVE] = ""
	}
	chatState.SavedButton = nil
	chatState.Ticket = database.Ticket{}

	return chatState.ChangeCache(cache, userID, lineID)
}

// сохранить данные о пользователе в кеше
func (chatState *Chat) SaveUserDataInCache(cl *client.Client, ctx context.Context, cache *bigcache.BigCache, userID, lineID uuid.UUID) (err error) {
	// получаем данные о пользователе
	userData, err := cl.GetSubscriber(ctx, userID)
	if err != nil {
		return
	}

	chatState.User = userData

	return chatState.ChangeCache(cache, userID, lineID)
}

// вернуться на предыдущий пункт меню в истории
func (chatState *Chat) HistoryStateBack(cache *bigcache.BigCache, userID, lineID uuid.UUID) error {
	// если истории нет то мы в старт должны быть
	if len(chatState.HistoryState) == 0 {
		chatState.PreviousState = database.GREETINGS
		return chatState.ChangeCache(cache, userID, lineID)
	}

	// удаление последнего меню в истории
	lastIndex := len(chatState.HistoryState) - 1
	chatState.HistoryState = chatState.HistoryState[:lastIndex]

	// если после удаления последнего меню не осталось истории
	if len(chatState.HistoryState) == 0 {
		chatState.PreviousState = database.GREETINGS
		return chatState.ChangeCache(cache, userID, lineID)
	}
	chatState.PreviousState = chatState.HistoryState[lastIndex-1]

	return chatState.ChangeCache(cache, userID, lineID)
}

// добавить новый пункт меню в историю
func (chatState *Chat) HistoryStateAppend(cache *bigcache.BigCache, userID, lineID uuid.UUID, state string) error {
	// чистим историю если меню последнее должно быть
	if slices.Contains([]string{database.FAIL_QNA, database.FINAL, database.START, database.GREETINGS}, state) {
		return chatState.HistoryStateClear(cache, userID, lineID)
	}

	// игнорируем добавление если спец кнопка
	if slices.Contains([]string{database.CREATE_TICKET, database.CREATE_TICKET_PREV_STAGE, database.WAIT_SEND}, state) {
		return nil
	}

	// если история пустая или последний элемент совпадает с переданным
	if len(chatState.HistoryState) == 0 || chatState.HistoryState[len(chatState.HistoryState)-1] != state {
		chatState.HistoryState = append(chatState.HistoryState, state)
	}

	return chatState.ChangeCache(cache, userID, lineID)
}

// очистить историю и необязательные поля
func (chatState *Chat) HistoryStateClear(cache *bigcache.BigCache, userID, lineID uuid.UUID) error {
	chatState.HistoryState = []string{}

	return chatState.ClearCacheOmitemptyFields(cache, userID, lineID)
}
