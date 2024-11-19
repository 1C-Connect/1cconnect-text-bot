package cache

import (
	"context"
	"encoding/json"
	"errors"

	"connect-text-bot/internal/botconfig_parser"
	"connect-text-bot/internal/connect/client"
	"connect-text-bot/internal/connect/response"
	"connect-text-bot/internal/database"
	"connect-text-bot/internal/logger"

	"github.com/allegro/bigcache/v3"
	"github.com/google/uuid"
)

func GetState(cl *client.Client, ctx context.Context, cache *bigcache.BigCache, userID, lineID uuid.UUID) Chat {
	var chatState Chat

	dbStateKey := userID.String() + ":" + lineID.String()

	b, err := cache.Get(dbStateKey)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			logger.Info("No state in cache for " + userID.String() + ":" + lineID.String())
			chatState = Chat{
				PreviousState: database.GREETINGS,
				CurrentState:  database.GREETINGS,
			}

			// сохраняем пользовательские данные
			err := chatState.SaveUserDataInCache(cl, ctx, cache, userID, lineID)
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

// получить данные пользователя
func (chatState *Chat) GetCacheUserInfo() response.User {
	return chatState.User
}

// получить значение переменной из хранимых данных
func (chatState *Chat) GetCacheVar(varName string) (string, bool) {
	result, exist := chatState.Vars[varName]
	return result, exist
}

// получить хранимые данные заявки
func (chatState *Chat) GetCacheTicket() database.Ticket {
	return chatState.Ticket
}

// получить хранимые данные заявки
func (chatState *Chat) GetCacheSavedButton() *botconfig_parser.Button {
	return chatState.SavedButton
}
