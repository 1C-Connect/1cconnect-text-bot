package bot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"connect-text-bot/bot/messages"
	"connect-text-bot/bot/requests"
	"connect-text-bot/botconfig_parser"
	"connect-text-bot/config"
	"connect-text-bot/database"
	"connect-text-bot/logger"

	"github.com/allegro/bigcache/v3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	// набор данных привязываемые к пользователю бота
	Chat struct {
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

func Receive(c *gin.Context) {
	var msg messages.Message
	if err := c.BindJSON(&msg); err != nil {
		logger.Warning("Error while receive message", err)

		c.Status(http.StatusBadRequest)
		return
	}

	logger.Debug("Receive message:", msg)

	// Реагируем только на сообщения пользователя
	if (msg.MessageType == messages.MESSAGE_TEXT || msg.MessageType == messages.MESSAGE_FILE) && msg.MessageAuthor != nil && msg.UserId != *msg.MessageAuthor {
		c.Status(http.StatusOK)
		return
	}

	cCp := c.Copy()
	go func(cCp *gin.Context, msg messages.Message) {
		chatState := getState(c, &msg)

		newState, err := processMessage(c, &msg, &chatState)
		if err != nil {
			logger.Warning("Error processMessage", err)
		}

		err = changeCacheState(c, &msg, &chatState, newState)
		if err != nil {
			logger.Warning("Error changeState", err)
		}
	}(cCp, msg)

	c.Status(http.StatusOK)
}

func changeCache(c *gin.Context, msg *messages.Message, chatState *Chat) error {
	cache := c.MustGet("cache").(*bigcache.BigCache)

	data, err := json.Marshal(chatState)
	if err != nil {
		logger.Warning("Error while change state to cache", err)
		return err
	}

	dbStateKey := msg.UserId.String() + ":" + msg.LineId.String()

	err = cache.Set(dbStateKey, data)
	logger.Debug("Write state to cache result")
	if err != nil {
		logger.Warning("Error while write state to cache", err)
	}

	return nil
}

func changeCacheTicket(c *gin.Context, msg *messages.Message, chatState *Chat, key string, value []string) error {
	t := database.Ticket{}

	switch key {
	case t.GetChannel(), t.GetTheme(), t.GetDescription():
		if len(value) != 1 {
			return errors.New(fmt.Sprint("Неверное количество аргументов для:", key))
		}
		switch key {
		case t.GetChannel():
			chatState.Ticket.ChannelID = uuid.MustParse(value[0])
		case t.GetTheme():
			chatState.Ticket.Theme = value[0]
		case t.GetDescription():
			chatState.Ticket.Description = value[0]
		}
	case t.GetExecutor(), t.GetService(), t.GetServiceType():
		if len(value) != 2 {
			return errors.New(fmt.Sprint("Неверное количество аргументов для:", key))
		}
		switch key {
		case t.GetExecutor():
			chatState.Ticket.Executor.Id = uuid.MustParse(value[0])
			chatState.Ticket.Executor.Name = value[1]
		case t.GetService():
			chatState.Ticket.Service.Id = uuid.MustParse(value[0])
			chatState.Ticket.Service.Name = value[1]
		case t.GetServiceType():
			chatState.Ticket.ServiceType.Id = uuid.MustParse(value[0])
			chatState.Ticket.ServiceType.Name = value[1]
		}
	}

	return changeCache(c, msg, chatState)
}

func changeCacheVars(c *gin.Context, msg *messages.Message, chatState *Chat, key, value string) error {
	if chatState.Vars == nil {
		chatState.Vars = make(map[string]string)
	}
	chatState.Vars[key] = value

	return changeCache(c, msg, chatState)
}

func changeCacheSavedButton(c *gin.Context, msg *messages.Message, chatState *Chat, button *botconfig_parser.Button) error {
	chatState.SavedButton = button

	return changeCache(c, msg, chatState)
}

func changeCacheState(c *gin.Context, msg *messages.Message, chatState *Chat, toState string) error {
	if chatState.CurrentState == toState {
		return nil
	}

	chatState.PreviousState = chatState.CurrentState
	chatState.CurrentState = toState

	return changeCache(c, msg, chatState)
}

func getState(c *gin.Context, msg *messages.Message) Chat {
	cache := c.MustGet("cache").(*bigcache.BigCache)

	var chatState Chat

	dbStateKey := msg.UserId.String() + ":" + msg.LineId.String()

	b, err := cache.Get(dbStateKey)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			logger.Info("No state in cache for " + msg.UserId.String() + ":" + msg.LineId.String())
			chatState = Chat{
				PreviousState: database.GREETINGS,
				CurrentState:  database.GREETINGS,
			}

			// сохраняем пользовательские данные
			err := saveUserDataInCache(c, msg, &chatState)
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
func getCacheVar(c *gin.Context, msg *messages.Message, varName string) (string, bool) {
	state := getState(c, msg)

	result, exist := state.Vars[varName]
	return result, exist
}

// чистим необязательные поля хранимых данных
func clearCacheOmitemptyFields(c *gin.Context, msg *messages.Message, chatState *Chat) error {
	if _, exist := chatState.Vars[database.VAR_FOR_SAVE]; exist {
		chatState.Vars[database.VAR_FOR_SAVE] = ""
	}
	chatState.SavedButton = nil
	chatState.Ticket = database.Ticket{}

	return changeCache(c, msg, chatState)
}

// сохранить данные о пользователе в кеше
func saveUserDataInCache(c *gin.Context, msg *messages.Message, chatState *Chat) (err error) {
	// получаем данные о пользователе
	userData, err := msg.GetSubscriber(c)
	if err != nil {
		return
	}

	chatState.User = userData
	return
}

// заполнить шаблон данными
func fillTemplateWithInfo(c *gin.Context, msg *messages.Message, text string) (result string, err error) {
	// проверяем есть ли шаблон в тексте чтобы лишний раз не выполнять обработку
	if !strings.Contains(text, "{{") && !strings.Contains(text, "}}") {
		return text, nil
	}

	state := getState(c, msg)

	// формируем шаблон
	templ, err := template.New("cmd").Parse(text)
	if err != nil {
		return
	}

	// создаем объединенные данные
	combinedData := struct {
		User   requests.User
		Var    map[string]string
		Ticket database.Ticket
	}{
		User:   state.User,
		Var:    state.Vars,
		Ticket: state.Ticket,
	}

	// заполняем шаблон
	var templOutput bytes.Buffer
	err = templ.Execute(&templOutput, combinedData)
	if err != nil {
		return
	}

	return templOutput.String(), err
}

// getFileNames - Получить список файлов из папки files.
func getFileNames(root string) map[string]bool {
	files := make(map[string]bool)
	root, _ = filepath.Abs(root)
	filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			files[path] = true
		}
		return nil
	})
	return files
}

func IsImage(file string) bool {
	reImg, _ := regexp.Compile(`(?i)\.(png|jpg|jpeg|bmp)$`)

	return reImg.MatchString(file)
}

func getFileInfo(filename, filesDir string) (isImage bool, filePath string, err error) {
	isImage = IsImage(filename)

	fileNames := getFileNames(filesDir)
	fullName, _ := filepath.Abs(filepath.Join(filesDir, filename))
	if !fileNames[fullName] {
		err = fmt.Errorf("не удалось найти и отправить файл: %s", filename)
		logger.Info(err)
	}

	filePath, _ = filepath.Abs(filepath.Join(filesDir, filename))
	return
}

func SendAnswer(c *gin.Context, msg *messages.Message, menu *botconfig_parser.Levels, goTo, filesDir string) {
	var toSend *[][]requests.KeyboardKey

	for i := 0; i < len(menu.Menu[goTo].Answer); i++ {
		// Отправляем клаву только с последним сообщением.
		// Т.к в дп4 криво отображается.
		if i == len(menu.Menu[goTo].Answer)-1 {
			toSend = menu.GenKeyboard(goTo)
		}
		if menu.Menu[goTo].Answer[i].Chat != "" {
			r, _ := fillTemplateWithInfo(c, msg, menu.Menu[goTo].Answer[i].Chat)
			msg.Send(c, r, toSend)
		}
		if menu.Menu[goTo].Answer[i].File != "" {
			if isImage, filePath, err := getFileInfo(menu.Menu[goTo].Answer[i].File, filesDir); err == nil {
				msg.SendFile(c, isImage, menu.Menu[goTo].Answer[i].File, filePath, &menu.Menu[goTo].Answer[i].FileText, toSend)
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// переход на следующую стадию формирования заявки
func nextStageTicketButton(c *gin.Context, msg *messages.Message, chatState *Chat, button *botconfig_parser.PartTicket, nextVar string, keyboard *[][]requests.KeyboardKey) (string, error) {
	goTo := database.CREATE_TICKET
	state := getState(c, msg)

	// формируем информацию о заявке
	tInfo, err := fillTemplateWithInfo(c, msg, state.SavedButton.TicketButton.TicketInfo)
	if err != nil {
		return finalSend(c, msg, "", err)
	}

	// формируем сообщение
	text := button.Text
	if button.DefaultValue != "" {
		text += fmt.Sprintf("\nЗначение по умолчанию: %s", button.DefaultValue)
	}
	r, err := fillTemplateWithInfo(c, msg, text)
	if err != nil {
		return finalSend(c, msg, "", err)
	}

	// сохраняем имя переменной куда будем записывать результат
	err = changeCacheVars(c, msg, chatState, database.VAR_FOR_SAVE, nextVar)
	if err != nil {
		return finalSend(c, msg, "", err)
	}

	// отображаем информацию
	err = msg.Send(c, tInfo, nil)
	if err != nil {
		return finalSend(c, msg, "", err)
	}
	err = msg.Send(c, r, keyboard)
	return goTo, err
}

// Проверить нажата ли BackButton
func getGoToIfClickedBackBtn(btn *botconfig_parser.Button, state Chat) (goTo string) {
	if btn != nil && btn.BackButton {
		if state.PreviousState != database.GREETINGS {
			goTo = state.PreviousState
		} else {
			goTo = database.START
		}
	}
	return
}

func processMessage(c *gin.Context, msg *messages.Message, chatState *Chat) (string, error) {
	cnf := c.MustGet("cnf").(*config.Conf)
	menu := c.MustGet("menus").(*botconfig_parser.Levels)
	time.Sleep(250 * time.Millisecond)

	var err error

	switch msg.MessageType {
	// Первый запуск.
	case messages.MESSAGE_TREATMENT_START_BY_USER:
		if chatState.CurrentState == database.START {
			return chatState.CurrentState, nil
		}
		return database.GREETINGS, nil

	// Нажатие меню ИЛИ Любое сообщение (текст, файл, попытка звонка).
	case messages.MESSAGE_CALL_START_TREATMENT,
		messages.MESSAGE_CALL_START_NO_TREATMENT,
		messages.MESSAGE_TREATMENT_START_BY_SPEC,
		messages.MESSAGE_TREATMENT_CLOSE,
		messages.MESSAGE_TREATMENT_CLOSE_ACTIVE:
		err = msg.Start(cnf)
		return database.GREETINGS, err

	case messages.MESSAGE_NO_FREE_SPECIALISTS:
		err = msg.RerouteTreatment(c)
		return database.GREETINGS, err

	// Пользователь отправил сообщение.
	case messages.MESSAGE_TEXT,
		messages.MESSAGE_FILE:
		text := strings.ToLower(strings.TrimSpace(msg.Text))

		switch chatState.CurrentState {

		case database.GREETINGS:
			switch text {
			case "меню", "menu":
				// Ходим дальше
			default:
				currentMenu := database.START
				if cm, ok := menu.Menu[currentMenu]; ok {
					if !cm.QnaDisable && menu.UseQNA.Enabled {
						// logger.Info("QNA", msg, chatState)
						qnaText, isClose, request_id, result_id := getMessageFromQNA(msg, cnf)
						if qnaText != "" {
							// Была подсказка
							go msg.QnaSelected(cnf, request_id, result_id)

							if isClose {
								err = msg.Send(c, qnaText, nil)
								msg.CloseTreatment(c)
								return currentMenu, err
							}

							err = msg.Send(c, qnaText, menu.GenKeyboard(currentMenu))
							return currentMenu, err
						}

						SendAnswer(c, msg, menu, database.FAIL_QNA, cnf.FilesDir)
						return database.FAIL_QNA, err
					}
				}
			}

			if menu.FirstGreeting {
				msg.Send(c, menu.GreetingMessage, nil)
				time.Sleep(time.Second)
			}
			SendAnswer(c, msg, menu, database.START, cnf.FilesDir)
			return database.START, err

		// пользователь попадет сюда в случае регистрации заявки
		case database.CREATE_TICKET:
			state := getState(c, msg)
			btn := menu.GetButton(state.CurrentState, text)
			tBtn := state.SavedButton.TicketButton
			ticket := database.Ticket{}

			// переходим если нажата BackButton
			goTo := getGoToIfClickedBackBtn(btn, state)
			if goTo != "" {
				// чистим данные
				err = clearCacheOmitemptyFields(c, msg, chatState)
				if err != nil {
					return finalSend(c, msg, "", err)
				}

				SendAnswer(c, msg, menu, goTo, cnf.FilesDir)
				return goTo, err
			}

			// узнаем имя переменной
			varName, exist := getCacheVar(c, msg, database.VAR_FOR_SAVE)
			if !exist {
				return finalSend(c, msg, "", err)
			}

			switch varName {
			case ticket.GetTheme():
				// если кнопка перехода к следующему шагу то пробуем поставить значение по умолчанию
				if btn != nil && btn.Goto == database.CREATE_TICKET {
					// если value есть то присвоить иначе сообщение пользователю
					if tBtn.Data.Theme.DefaultValue != "" {
						// подставляем данные если value содержит шаблон
						defaultValue, err := fillTemplateWithInfo(c, msg, tBtn.Data.Theme.DefaultValue)
						if err != nil {
							return finalSend(c, msg, "", err)
						}

						err = changeCacheTicket(c, msg, chatState, varName, []string{defaultValue})
						if err != nil {
							return finalSend(c, msg, "", err)
						}
					} else {
						err = msg.Send(c, "Данный этап нельзя пропустить тк отсутствует значение по умолчанию", nil)
						return database.CREATE_TICKET, err
					}
				} else {
					err = changeCacheTicket(c, msg, chatState, varName, []string{msg.Text})
					if err != nil {
						return finalSend(c, msg, "", err)
					}
				}

				return nextStageTicketButton(c, msg, chatState, tBtn.Data.Description, ticket.GetDescription(), menu.GenKeyboard(database.CREATE_TICKET))

			case ticket.GetDescription():
				// если кнопка перехода к следующему шагу то пробуем поставить значение по умолчанию
				if btn != nil && btn.Goto == database.CREATE_TICKET {
					// если value есть то присвоить иначе сообщение пользователю
					if tBtn.Data.Description.DefaultValue != "" {
						// подставляем данные если value содержит шаблон
						defaultValue, err := fillTemplateWithInfo(c, msg, tBtn.Data.Description.DefaultValue)
						if err != nil {
							return finalSend(c, msg, "", err)
						}

						err = changeCacheTicket(c, msg, chatState, varName, []string{defaultValue})
						if err != nil {
							return finalSend(c, msg, "", err)
						}
					} else {
						err = msg.Send(c, "Данный этап нельзя пропустить тк отсутствует значение по умолчанию", nil)
						return database.CREATE_TICKET, err
					}
				} else {
					err = changeCacheTicket(c, msg, chatState, varName, []string{msg.Text})
					if err != nil {
						return finalSend(c, msg, "", err)
					}
				}

				// получаем список специалистов
				listSpecs, err := msg.GetSpecialists(c, msg.LineId)
				if err != nil {
					return finalSend(c, msg, "", err)
				}

				// формируем клавиатуру
				answer := menu.GenKeyboard(database.CREATE_TICKET)
				for _, v := range listSpecs {
					*answer = append(*answer, []requests.KeyboardKey{{Text: fmt.Sprintf("%s %s %s", v.Surname, v.Name, v.Patronymic)}})
				}

				return nextStageTicketButton(c, msg, chatState, tBtn.Data.Executor, ticket.GetExecutor(), answer)

			case ticket.GetExecutor():
				// получаем список специалистов
				listSpecs, err := msg.GetSpecialists(c, msg.LineId)
				if err != nil {
					return finalSend(c, msg, "", err)
				}

				// функция для обработки найденного значения
				checkValue := func(checkValue string) (_ string, err error) {
					isFind := false
					for _, v := range listSpecs {
						fio := fmt.Sprintf("%s %s %s", v.Surname, v.Name, v.Patronymic)
						if checkValue == strings.TrimSpace(fio) {
							err = changeCacheTicket(c, msg, chatState, varName, []string{v.UserId.String(), checkValue})
							if err != nil {
								return finalSend(c, msg, "", err)
							}

							isFind = true
							break
						}
					}
					if !isFind {
						err = msg.Send(c, "Получено некорректное значение. Повторите попытку", nil)
						return database.CREATE_TICKET, err
					}
					return
				}

				// если кнопка перехода к следующему шагу то пробуем поставить значение по умолчанию
				if btn != nil && btn.Goto == database.CREATE_TICKET {
					// если value есть то присвоить иначе сообщение пользователю
					if tBtn.Data.Executor.DefaultValue != "" {
						// подставляем данные если value содержит шаблон
						defaultValue, err := fillTemplateWithInfo(c, msg, tBtn.Data.Executor.DefaultValue)
						if err != nil {
							return finalSend(c, msg, "", err)
						}

						if goTo, err := checkValue(defaultValue); goTo != "" || err != nil {
							return goTo, err
						}
					} else {
						err = msg.Send(c, "Данный этап нельзя пропустить тк отсутствует значение по умолчанию", nil)
						return database.CREATE_TICKET, err
					}
				} else {
					// ищем что было выбрано
					if goTo, err := checkValue(msg.Text); goTo != "" || err != nil {
						return goTo, err
					}
				}

				// формируем клавиатуру
				answer := menu.GenKeyboard(database.CREATE_TICKET)
				kinds, err := msg.GetTicketDataKinds(c, nil)
				if err != nil {
					return finalSend(c, msg, "", err)
				}
				for _, v := range kinds {
					*answer = append(*answer, []requests.KeyboardKey{{Text: v.Name}})
				}

				return nextStageTicketButton(c, msg, chatState, tBtn.Data.Service, ticket.GetService(), answer)

			case ticket.GetService():
				// получаем данные для заявок
				ticketData, err := msg.GetTicketData(c)
				if err != nil {
					return finalSend(c, msg, "", err)
				}
				kinds, err := msg.GetTicketDataKinds(c, ticketData)
				if err != nil {
					return finalSend(c, msg, "", err)
				}

				// переменная для сохранения какой вид услуги был выбран
				selectedKind := requests.TicketDataKind{}

				// функция для обработки найденного значения
				checkValue := func(checkValue string) (_ string, err error) {
					isFind := false
					for _, v := range kinds {
						if checkValue == v.Name {
							err = changeCacheTicket(c, msg, chatState, varName, []string{v.ID.String(), checkValue})
							if err != nil {
								return finalSend(c, msg, "", err)
							}

							selectedKind = v
							isFind = true
							break
						}
					}
					if !isFind {
						err = msg.Send(c, "Получено некорректное значение. Повторите попытку", nil)
						return database.CREATE_TICKET, err
					}
					return
				}

				// если кнопка перехода к следующему шагу то пробуем поставить значение по умолчанию
				if btn != nil && btn.Goto == database.CREATE_TICKET {
					// если value есть то присвоить иначе сообщение пользователю
					if tBtn.Data.Service.DefaultValue != "" {
						// подставляем данные если value содержит шаблон
						defaultValue, err := fillTemplateWithInfo(c, msg, tBtn.Data.Service.DefaultValue)
						if err != nil {
							return finalSend(c, msg, "", err)
						}

						// ищем что было выбрано
						if goTo, err := checkValue(defaultValue); goTo != "" || err != nil {
							return goTo, err
						}
					} else {
						err = msg.Send(c, "Данный этап нельзя пропустить тк отсутствует значение по умолчанию", nil)
						return database.CREATE_TICKET, err
					}
				} else {
					// ищем что было выбрано
					if goTo, err := checkValue(msg.Text); goTo != "" || err != nil {
						return goTo, err
					}
				}

				// формируем клавиатуру
				answer := menu.GenKeyboard(database.CREATE_TICKET)
				kindTypes, err := msg.GetTicketDataTypesWhereKind(c, ticketData, selectedKind.ID)
				if err != nil {
					return finalSend(c, msg, "", err)
				}
				for _, v := range kindTypes {
					*answer = append(*answer, []requests.KeyboardKey{{Text: v.Name}})
				}

				return nextStageTicketButton(c, msg, chatState, tBtn.Data.ServiceType, ticket.GetServiceType(), answer)

			case ticket.GetServiceType():
				types, err := msg.GetTicketDataTypesWhereKind(c, nil, state.Ticket.Service.Id)
				if err != nil {
					return finalSend(c, msg, "", err)
				}

				// функция для обработки найденного значения
				checkValue := func(checkValue string) (_ string, err error) {
					isFind := false
					for _, v := range types {
						if checkValue == v.Name {
							err = changeCacheTicket(c, msg, chatState, varName, []string{v.ID.String(), checkValue})
							if err != nil {
								return finalSend(c, msg, "", err)
							}

							isFind = true
							break
						}
					}
					if !isFind {
						err = msg.Send(c, "Получено некорректное значение. Повторите попытку", nil)
						return database.CREATE_TICKET, err
					}
					return
				}

				// если кнопка перехода к следующему шагу то пробуем поставить значение по умолчанию
				if btn != nil && btn.Goto == database.CREATE_TICKET {
					// если value есть то присвоить иначе сообщение пользователю
					if tBtn.Data.ServiceType.DefaultValue != "" {
						// подставляем данные если value содержит шаблон
						defaultValue, err := fillTemplateWithInfo(c, msg, tBtn.Data.ServiceType.DefaultValue)
						if err != nil {
							return finalSend(c, msg, "", err)
						}

						// ищем что было выбрано
						if goTo, err := checkValue(defaultValue); goTo != "" || err != nil {
							return goTo, err
						}
					} else {
						err = msg.Send(c, "Данный этап нельзя пропустить тк отсутствует значение по умолчанию", nil)
						return database.CREATE_TICKET, err
					}
				} else {
					// ищем что было выбрано
					if goTo, err := checkValue(msg.Text); goTo != "" || err != nil {
						return goTo, err
					}
				}

				// переходим в финальный этап формирования заявки
				err = changeCacheVars(c, msg, chatState, database.VAR_FOR_SAVE, "FINAL")
				if err != nil {
					return finalSend(c, msg, "", err)
				}

				// отображаем информацию о заявке
				tInfo, err := fillTemplateWithInfo(c, msg, state.SavedButton.TicketButton.TicketInfo)
				if err != nil {
					return finalSend(c, msg, "", err)
				}
				err = msg.Send(c, tInfo, nil)
				if err != nil {
					return finalSend(c, msg, "", err)
				}

				err = msg.Send(c, "Зарегистрировать заявку?", menu.GenKeyboard(database.CREATE_TICKET))
				return database.CREATE_TICKET, err

			// этап регистрации заявки
			case "FINAL":
				if btn != nil && btn.Goto == database.CREATE_TICKET {
					err = msg.DropKeyboard(c)
					if err != nil {
						return finalSend(c, msg, "", err)
					}

					err = msg.Send(c, "Идет процесс регистрации заявки, подождите немного", nil)
					if err != nil {
						return finalSend(c, msg, "", err)
					}

					// регистрируем заявку
					r, err := msg.ServiceRequestAdd(c, state.Ticket)
					if err != nil {
						return finalSend(c, msg, "", err)
					}

					// даем время чтобы загрузилась заявка
					for {
						time.Sleep(4 * time.Second)

						_, err := msg.GetTicket(c, uuid.MustParse(r["ServiceRequestID"]))
						if err == nil {
							break
						}
					}

					// чистим данные
					err = clearCacheOmitemptyFields(c, msg, chatState)
					if err != nil {
						return finalSend(c, msg, "", err)
					}

					SendAnswer(c, msg, menu, tBtn.Goto, cnf.FilesDir)
					return tBtn.Goto, err
				}
			}

			err = msg.Send(c, "Ожидалось нажатие на кнопку. Повторите попытку", nil)
			return database.CREATE_TICKET, err

		// пользователь попадет сюда в случае перехода в режим ожидания сообщения
		case database.WAIT_SEND:
			state := getState(c, msg)

			// записываем введенные данные в переменную
			varName, ok := getCacheVar(c, msg, database.VAR_FOR_SAVE)
			if ok && varName != "" {
				changeCacheVars(c, msg, chatState, varName, msg.Text)
			}

			// переходим если нажата BackButton
			btn := menu.GetButton(state.CurrentState, text)
			goTo := getGoToIfClickedBackBtn(btn, state)
			if goTo != "" {
				// чистим данные
				err = clearCacheOmitemptyFields(c, msg, chatState)
				if err != nil {
					return finalSend(c, msg, "", err)
				}

				SendAnswer(c, msg, menu, goTo, cnf.FilesDir)
				return goTo, err
			}

			// выполнить действие кнопки
			err = changeCacheState(c, msg, chatState, database.START)
			if err != nil {
				logger.Warning("Error changeState", err)
			}
			return processMessage(c, msg, chatState)

		default:
			state := getState(c, msg)
			currentMenu := state.CurrentState

			// В редисе может остаться состояние которого, нет в конфиге.
			cm, ok := menu.Menu[currentMenu]
			if !ok {
				logger.Warning("неизвестное состояние: ", currentMenu)
				err = msg.Send(c, menu.ErrorMessage, menu.GenKeyboard(database.START))
				return database.GREETINGS, err
			}

			// определяем какая кнопка была нажата
			btn := menu.GetButton(currentMenu, text)
			if btn == nil {
				text = strings.ReplaceAll(text, "«", "\"")
				text = strings.ReplaceAll(text, "»", "\"")
				btn = menu.GetButton(currentMenu, text)
			}

			// если есть принудительное значение для кнопки то присвоить
			if state.SavedButton != nil {
				btn = state.SavedButton

				// очищаем данные чтобы не было повторного использования
				err = clearCacheOmitemptyFields(c, msg, chatState)
				if err != nil {
					return finalSend(c, msg, "", err)
				}
			}

			if btn != nil {
				goTo := btn.Goto
				if btn.BackButton {
					if state.PreviousState != database.GREETINGS {
						goTo = state.PreviousState
					} else {
						goTo = database.START
					}
				}

				for i := 0; i < len(btn.Chat); i++ {
					if btn.Chat[i].Chat != "" && !btn.CloseButton && !btn.RedirectButton {
						r, err := fillTemplateWithInfo(c, msg, btn.Chat[i].Chat)
						if err != nil {
							return finalSend(c, msg, "", err)
						}

						msg.Send(c, r, nil)
					}
					if btn.Chat[i].File != "" {
						if isImage, filepath, err := getFileInfo(btn.Chat[i].File, cnf.FilesDir); err == nil {
							err := msg.SendFile(c, isImage, btn.Chat[i].File, filepath, &btn.Chat[i].FileText, nil)
							if err != nil {
								logger.Warning(err)
							}
						}
					}
					time.Sleep(250 * time.Millisecond)
				}
				if btn.CloseButton {
					// чистим данные
					err = clearCacheOmitemptyFields(c, msg, chatState)
					if err != nil {
						return finalSend(c, msg, "", err)
					}

					err = msg.CloseTreatment(c)
					return database.GREETINGS, err
				}
				if btn.RedirectButton {
					err = msg.RerouteTreatment(c)
					return database.GREETINGS, err
				}
				if btn.AppointSpecButton != nil && *btn.AppointSpecButton != uuid.Nil {
					ok, err := msg.GetSpecialistAvailable(c, *btn.AppointSpecButton)
					if err != nil || !ok {
						return finalSend(c, msg, "Выбранный специалист недоступен", err)
					}
					err = msg.AppointSpec(c, *btn.AppointSpecButton)
					return database.GREETINGS, err
				}
				if btn.AppointRandomSpecFromListButton != nil && len(*btn.AppointRandomSpecFromListButton) != 0 {
					// получаем список свободных специалистов
					r, err := msg.GetSpecialistsAvailable(c)
					if err != nil || len(r) == 0 {
						return finalSend(c, msg, "Специалисты данной области недоступны", err)
					}

					// создаем словарь id специалистов которых мы хотели бы назначить
					specIDs := make(map[uuid.UUID]struct{})
					for _, id := range *btn.AppointRandomSpecFromListButton {
						specIDs[id] = struct{}{}
					}

					// ищем среди свободных специалистов нужных
					neededSpec := make([]uuid.UUID, 0)
					for i := 0; i < len(r); i++ {
						if _, exists := specIDs[r[i]]; exists {
							neededSpec = append(neededSpec, r[i])
						}
					}

					// проверяем есть ли хотя бы 1 свободный специалист
					lenNeededSpec := len(neededSpec)
					if lenNeededSpec == 0 {
						return finalSend(c, msg, "Специалисты данной области недоступны", err)
					}

					// назначаем случайного специалиста из списка
					seed := time.Now().UnixNano()
					rns := rand.NewSource(seed)
					rng := rand.New(rns)
					randomIndex := rng.Intn(lenNeededSpec)
					err = msg.AppointSpec(c, neededSpec[randomIndex])
					return database.GREETINGS, err
				}
				if btn.RerouteButton != nil && *btn.RerouteButton != uuid.Nil {
					r, err := msg.GetSubscriptions(c, *btn.RerouteButton)
					if err != nil {
						return finalSend(c, msg, "", err)
					}
					if len(r) == 0 {
						return finalSend(c, msg, "Выбранная линия недоступна", err)
					}

					err = msg.Reroute(c, *btn.RerouteButton, "")
					if err != nil {
						return finalSend(c, msg, "", err)
					}
					return database.GREETINGS, err
				}
				if btn.ExecButton != "" {
					r, err := fillTemplateWithInfo(c, msg, btn.ExecButton)
					if err != nil {
						return finalSend(c, msg, "", err)
					}

					// выполняем команду на устройстве
					cmdParts := strings.Fields(r)
					var cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
					cmdOutput, err := cmd.CombinedOutput()
					if err != nil {
						return finalSend(c, msg, "Ошибка: "+err.Error(), err)
					}

					// выводим результат и завершаем
					msg.Send(c, string(cmdOutput), nil)
					goTo := database.FINAL
					if btn.Goto != "" {
						goTo = btn.Goto
					}
					SendAnswer(c, msg, menu, goTo, cnf.FilesDir)
					return goTo, err
				}
				if btn.SaveToVar != nil {
					// Сообщаем пользователю что требуем и запускаем ожидание данных
					goTo := database.WAIT_SEND
					if btn.SaveToVar.SendText != nil && *btn.SaveToVar.SendText != "" {
						r, err := fillTemplateWithInfo(c, msg, *btn.SaveToVar.SendText)
						if err != nil {
							return finalSend(c, msg, "", err)
						}

						msg.Send(c, r, menu.GenKeyboard(goTo))
					} else {
						// выводим default WAIT_SEND меню в случае отсутствия настроек текста
						SendAnswer(c, msg, menu, goTo, cnf.FilesDir)
					}

					// сохраняем имя переменной куда будем записывать результат
					err := changeCacheVars(c, msg, chatState, database.VAR_FOR_SAVE, btn.SaveToVar.VarName)
					if err != nil {
						return finalSend(c, msg, "", err)
					}

					// сохраняем ссылку на кнопку которая будет выполнена после завершения
					if btn.SaveToVar.DoButton != nil {
						err = changeCacheSavedButton(c, msg, chatState, btn.SaveToVar.DoButton)
						if err != nil {
							return finalSend(c, msg, "", err)
						}
					}

					return goTo, err
				}
				if btn.TicketButton != nil {
					// сохраняем ссылку на кнопку которая была нажата
					err = changeCacheSavedButton(c, msg, chatState, btn)
					if err != nil {
						return finalSend(c, msg, "", err)
					}

					t := database.Ticket{}

					// сохраняем id канала поступления заявки
					err = changeCacheTicket(c, msg, chatState, t.GetChannel(), []string{btn.TicketButton.ChannelID.String()})
					if err != nil {
						return finalSend(c, msg, "", err)
					}

					return nextStageTicketButton(c, msg, chatState, btn.TicketButton.Data.Theme, t.GetTheme(), menu.GenKeyboard(database.CREATE_TICKET))
				}

				// Сообщения при переходе на новое меню.
				SendAnswer(c, msg, menu, goTo, cnf.FilesDir)
				return goTo, err

			} else { // Произвольный текст
				if !cm.QnaDisable && menu.UseQNA.Enabled {
					// logger.Info("QNA", msg, chatState)

					qnaText, isClose, request_id, result_id := getMessageFromQNA(msg, cnf)
					if qnaText != "" {
						// Была подсказка
						go msg.QnaSelected(cnf, request_id, result_id)

						if isClose {
							err = msg.Send(c, qnaText, nil)
							msg.CloseTreatment(c)
							return state.CurrentState, err
						}

						err = msg.Send(c, qnaText, menu.GenKeyboard(currentMenu))
						return state.CurrentState, err
					}

					SendAnswer(c, msg, menu, database.FAIL_QNA, cnf.FilesDir)
					return database.FAIL_QNA, err
				}
				err = msg.Send(c, menu.ErrorMessage, menu.GenKeyboard(currentMenu))
				return state.CurrentState, err
			}
		}
	case messages.MESSAGE_TREATMENT_TO_BOT:
	default:
		panic(fmt.Sprintf("unexpected messages.MessageType: %#v", msg.MessageType))
	}
	return database.GREETINGS, errors.New("i don't know what i should do")
}

// выполнить Send и вывести Final меню
func finalSend(c *gin.Context, msg *messages.Message, finalMsg string, err error) (string, error) {
	cnf := c.MustGet("cnf").(*config.Conf)
	menu := c.MustGet("menus").(*botconfig_parser.Levels)

	if finalMsg != "" {
		msg.Send(c, finalMsg, nil)
	} else {
		msg.Send(c, "Во время обработки вашего запроса произошла ошибка", nil)
	}
	goTo := database.FINAL
	SendAnswer(c, msg, menu, goTo, cnf.FilesDir)
	return goTo, err
}

// getMessageFromQNA - Метод возвращает ответ с Базы Знаний, и флаг, если это сообщение закрывает обращение.
func getMessageFromQNA(msg *messages.Message, cnf *config.Conf) (string, bool, uuid.UUID, uuid.UUID) {
	result_id := uuid.Nil
	qnaAnswer := msg.GetQNA(cnf, false, false)

	for _, v := range qnaAnswer.Answers {
		if v.Accuracy > 0 {
			result_id = v.ID
			return v.Text, v.AnswerSource == "GOODBYES", qnaAnswer.RequestID, result_id
		}
	}

	return "", false, result_id, result_id
}
