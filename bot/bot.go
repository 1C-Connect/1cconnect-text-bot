package bot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"math/rand"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"connect-text-bot/internal/botconfig_parser"
	"connect-text-bot/internal/cache"
	"connect-text-bot/internal/config"
	"connect-text-bot/internal/connect/messages"
	"connect-text-bot/internal/connect/requests"
	"connect-text-bot/internal/connect/response"
	"connect-text-bot/internal/database"
	"connect-text-bot/internal/logger"
	"connect-text-bot/internal/us"

	"github.com/allegro/bigcache/v3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hooklift/gowsdl/soap"
	"github.com/kballard/go-shellquote"
)

type MultiData struct {
	cacheDB    *bigcache.BigCache
	soapcl     *soap.Client
	soapclmtom *soap.Client
	cnf        *config.Conf
	menu       *botconfig_parser.Levels
	bot        Bot
	msg        messages.Message
	chatState  *cache.Chat
}

func Receive(c *gin.Context) {
	cacheDB := c.MustGet("cache").(*bigcache.BigCache)
	soapcl := c.MustGet("soapcl").(*soap.Client)
	soapclmtom := c.MustGet("soapclmtom").(*soap.Client)
	cnf := c.MustGet("cnf").(*config.Conf)
	menus := c.MustGet("menus").(*botconfig_parser.Levels)

	var msg messages.Message
	if err := c.BindJSON(&msg); err != nil {
		logger.Warning("Error while receive message", err)

		c.Status(http.StatusBadRequest)
		return
	}

	logger.Debug("Receive message:", msg)

	// Реагируем только на сообщения пользователя
	if (msg.MessageType == messages.MESSAGE_TEXT || msg.MessageType == messages.MESSAGE_FILE) && msg.MessageAuthor != nil && msg.UserID != *msg.MessageAuthor {
		c.Status(http.StatusOK)
		return
	}

	bot, ok := botsConnect[msg.LineID]
	if !ok {
		logger.Warning("request not find. line_id=", msg.LineID.String())
		return
	}

	go func() {
		chatState := cache.GetState(bot.connect, c, cacheDB, msg.UserID, msg.LineID)

		md := MultiData{
			cacheDB:    cacheDB,
			soapcl:     soapcl,
			soapclmtom: soapclmtom,
			cnf:        cnf,
			menu:       menus,
			bot:        bot,
			msg:        msg,
			chatState:  &chatState,
		}

		newState, err := processMessage(&md)
		if err != nil {
			logger.Warning("Error processMessage", err)
			return
		}

		err = md.chatState.ChangeCacheState(cacheDB, msg.UserID, msg.LineID, newState)
		if err != nil {
			logger.Warning("Error changeState", err)
		}

		logger.Debug("Cache:", chatState)
	}()

	c.Status(http.StatusOK)
}

// заполнить шаблон данными
func fillTemplateWithInfo(state *cache.Chat, text string) (result string, err error) {
	// проверяем есть ли шаблон в тексте чтобы лишний раз не выполнять обработку
	if !strings.Contains(text, "{{") || !strings.Contains(text, "}}") {
		return text, nil
	}

	// формируем шаблон
	templ, err := template.New("cmd").Parse(text)
	if err != nil {
		return
	}

	// создаем объединенные данные
	combinedData := struct {
		User   response.User
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
	_ = filepath.Walk(root, func(path string, info fs.FileInfo, _ error) error {
		// проверяем, что текущий элемент не является директорией
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
	fullName, _ := filepath.Abs(filepath.Join(filesDir, filename))

	// проверяем есть ли файл в указанном месте
	fileNames := getFileNames(filesDir)
	if !fileNames[fullName] {
		err = fmt.Errorf("не удалось найти и отправить файл: %s", filename)
		logger.Info(err)
	}

	return IsImage(filename), fullName, err
}

// отправить сообщение из меню
func SendAnswerMenuChat(ctx context.Context, md *MultiData, answer *botconfig_parser.Answer, keyboard *[][]requests.KeyboardKey) error {
	if answer.Chat != "" {
		r, err := fillTemplateWithInfo(md.chatState, answer.Chat)
		if err != nil {
			return err
		}
		err = md.bot.connect.Send(ctx, md.msg.UserID, r, keyboard)
		return err
	}
	return nil
}

// отправить файл из меню
func SendAnswerMenuFile(ctx context.Context, md *MultiData, answer *botconfig_parser.Answer, keyboard *[][]requests.KeyboardKey) {
	if answer.File != "" {
		if isImage, filePath, err := getFileInfo(answer.File, md.cnf.FilesDir); err == nil {
			err = md.bot.connect.SendFile(ctx, md.msg.UserID, isImage, answer.File, filePath, &answer.FileText, keyboard)
			if err != nil {
				logger.Warning(err)
			}
		} else {
			_ = md.bot.connect.Send(ctx, md.msg.UserID, md.menu.ErrorMessages.FailedSendFile, keyboard)
		}
	}
}

// отобразить настройки меню
func SendAnswerMenu(ctx context.Context, md *MultiData, answer []*botconfig_parser.Answer, keyboard *[][]requests.KeyboardKey) error {
	var toSend *[][]requests.KeyboardKey

	for i := range len(answer) {
		// Отправляем клаву только с последним сообщением.
		// Т.к в дп4 криво отображается.
		if i == len(answer)-1 {
			toSend = keyboard
		}
		err := SendAnswerMenuChat(ctx, md, answer[i], toSend)
		if err != nil {
			return err
		}
		SendAnswerMenuFile(ctx, md, answer[i], toSend)
		time.Sleep(250 * time.Millisecond)
	}
	return nil
}

// отобразить меню и выполнить do_button если есть
func SendAnswer(ctx context.Context, md *MultiData, goTo string, err error) (string, error) {
	errMenu := SendAnswerMenu(ctx, md, md.menu.Menu[goTo].Answer, md.menu.GenKeyboard(goTo))
	if errMenu != nil {
		return finalSend(ctx, md, "", err)
	}

	// выполнить действие do_button если не было ошибок и есть такая настройка
	if err == nil && md.menu.Menu[goTo].DoButton != nil {
		if md.menu.Menu[goTo].DoButton.NestedMenu != nil {
			return SendAnswer(ctx, md, md.menu.Menu[goTo].DoButton.NestedMenu.ID, err)
		}

		gt, err := triggerButton(ctx, md, md.menu.Menu[goTo].DoButton)
		_ = md.chatState.HistoryStateAppend(md.cacheDB, md.msg.UserID, md.msg.LineID, gt)
		return gt, err
	}
	return goTo, err
}

// переход на следующую стадию формирования заявки
func nextStageTicketButton(ctx context.Context, md *MultiData, button *botconfig_parser.TicketButton, nextVar string) (string, error) {
	ticket := database.Ticket{}
	text := ""

	chatState, msg, bot := md.chatState, md.msg, md.bot

	// настройки для клавиатуры
	keyboard := &[][]requests.KeyboardKey{}
	btnAgain := []requests.KeyboardKey{{ID: "1", Text: "Далее"}}
	btnBack := []requests.KeyboardKey{{ID: "2", Text: "Назад"}}
	btnCancel := []requests.KeyboardKey{{ID: "0", Text: "Отмена"}}
	btnConfirm := []requests.KeyboardKey{{ID: "1", Text: "Подтверждаю"}}

	// проверяем на какое следующее меню надо отправить
	if nextVar == ticket.GetTheme() {
		text = button.Data.Theme.Text
		if button.Data.Theme.DefaultValue != nil {
			// подставляем данные если value содержит шаблон
			defaultValue, err := fillTemplateWithInfo(chatState, *button.Data.Theme.DefaultValue)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			// присвоить значение по умолчанию
			err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, nextVar, database.TicketPart{Name: &defaultValue})
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			// переходим к следующее шагу
			nextVar = ticket.GetDescription()
		} else {
			// формируем клавиатуру
			*keyboard = append(*keyboard, btnAgain)
			*keyboard = append(*keyboard, btnBack)
			*keyboard = append(*keyboard, btnCancel)
		}
	}
	if nextVar == ticket.GetDescription() {
		text = button.Data.Description.Text
		if button.Data.Description.DefaultValue != nil {
			// подставляем данные если value содержит шаблон
			defaultValue, err := fillTemplateWithInfo(chatState, *button.Data.Description.DefaultValue)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			// присвоить значение по умолчанию
			err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, nextVar, database.TicketPart{Name: &defaultValue})
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			// переходим к следующее шагу
			nextVar = ticket.GetExecutor()
		} else {
			// формируем клавиатуру
			*keyboard = append(*keyboard, btnAgain)
			*keyboard = append(*keyboard, btnBack)
			*keyboard = append(*keyboard, btnCancel)
		}
	}
	if nextVar == ticket.GetExecutor() {
		text = button.Data.Executor.Text
		if button.Data.Executor.DefaultValue != nil {
			r, err := bot.connect.GetSpecialist(ctx, uuid.MustParse(*button.Data.Executor.DefaultValue))
			if err != nil {
				return finalSend(ctx, md, "", err)
			}
			fio := strings.TrimSpace(fmt.Sprintf("%s %s %s", r.Surname, r.Name, r.Patronymic))

			// присвоить значение по умолчанию
			err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, nextVar, database.TicketPart{ID: r.UserID, Name: &fio})
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			// переходим к следующее шагу
			nextVar = ticket.GetService()
		} else {
			// получаем список специалистов
			listSpecs, err := bot.connect.GetSpecialists(ctx, msg.LineID)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			// формируем клавиатуру
			for _, v := range listSpecs {
				*keyboard = append(*keyboard, []requests.KeyboardKey{{Text: fmt.Sprintf("%s %s %s", v.Surname, v.Name, v.Patronymic)}})
			}
			*keyboard = append(*keyboard, btnAgain)
			*keyboard = append(*keyboard, btnBack)
			*keyboard = append(*keyboard, btnCancel)
		}
	}
	var ticketData *response.GetTicketDataResponse
	if nextVar == ticket.GetService() {
		text = button.Data.Service.Text
		if button.Data.Service.DefaultValue != nil {
			tr, err := bot.connect.GetTicketData(ctx, chatState.User.CounterpartOwnerID)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}
			ticketData = &tr
			kinds, err := bot.connect.GetTicketDataKinds(ctx, ticketData, chatState.User.CounterpartOwnerID)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			// присвоить значение по умолчанию
			isFind := false
			for _, k := range kinds {
				if k.ID.String() == *button.Data.Service.DefaultValue {
					err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, nextVar, database.TicketPart{ID: k.ID, Name: &k.Name})
					if err != nil {
						return finalSend(ctx, md, "", err)
					}
					isFind = true
					break
				}
			}
			if !isFind {
				return finalSend(ctx, md, "", errors.New("указанное значение по умолчанию (value) невозможно применить в (service) по текущей линии"))
			}

			// переходим к следующее шагу
			nextVar = ticket.GetServiceType()
		} else {
			// формируем клавиатуру
			kinds, err := bot.connect.GetTicketDataKinds(ctx, nil, chatState.User.CounterpartOwnerID)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}
			for _, v := range kinds {
				*keyboard = append(*keyboard, []requests.KeyboardKey{{Text: v.Name}})
			}
			*keyboard = append(*keyboard, btnAgain)
			*keyboard = append(*keyboard, btnBack)
			*keyboard = append(*keyboard, btnCancel)
		}
	}
	if nextVar == ticket.GetServiceType() {
		text = button.Data.ServiceType.Text
		if button.Data.ServiceType.DefaultValue != nil {
			types, err := bot.connect.GetTicketDataTypesWhereKind(ctx, ticketData, chatState.User.CounterpartOwnerID, chatState.Ticket.Service.ID)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			// присвоить значение по умолчанию
			isFind := false
			for _, t := range types {
				if t.ID.String() == *button.Data.ServiceType.DefaultValue {
					err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, nextVar, database.TicketPart{ID: t.ID, Name: &t.Name})
					if err != nil {
						return finalSend(ctx, md, "", err)
					}
					isFind = true
					break
				}
			}
			if !isFind {
				return finalSend(ctx, md, "", errors.New("указанное значение по умолчанию (value) невозможно применить в (type) для выбранного (service)"))
			}

			// переходим к следующее шагу
			nextVar = ticket.GetFinal()
		} else {
			// формируем клавиатуру
			kindTypes, err := bot.connect.GetTicketDataTypesWhereKind(ctx, nil, chatState.User.CounterpartOwnerID, chatState.Ticket.Service.ID)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}
			for _, v := range kindTypes {
				*keyboard = append(*keyboard, []requests.KeyboardKey{{Text: v.Name}})
			}
			*keyboard = append(*keyboard, btnAgain)
			*keyboard = append(*keyboard, btnBack)
			*keyboard = append(*keyboard, btnCancel)
		}
	}
	if nextVar == ticket.GetFinal() {
		text = button.TicketInfo
		*keyboard = append(*keyboard, btnConfirm)
		*keyboard = append(*keyboard, btnBack)
		*keyboard = append(*keyboard, btnCancel)
	}

	// формируем сообщение
	r, err := fillTemplateWithInfo(chatState, text)
	if err != nil {
		return finalSend(ctx, md, "", err)
	}

	// сохраняем имя переменной куда будем записывать результат
	_ = chatState.ChangeCacheVars(md.cacheDB, msg.UserID, msg.LineID, database.VAR_FOR_SAVE, nextVar)

	err = bot.connect.Send(ctx, msg.UserID, r, keyboard)
	return database.CREATE_TICKET, err
}

// возврат на предыдущий шаг формирования заявки
func prevStageTicketButton(ctx context.Context, md *MultiData, button *botconfig_parser.TicketButton, currentVar string) (string, error) {
	t := database.Ticket{}

	var stages = []struct {
		getCurrent   func() string
		getNext      func() string
		defaultValue *string
	}{
		{t.GetFinal, t.GetServiceType, button.Data.ServiceType.DefaultValue},
		{t.GetServiceType, t.GetService, button.Data.Service.DefaultValue},
		{t.GetService, t.GetExecutor, button.Data.Executor.DefaultValue},
		{t.GetExecutor, t.GetDescription, button.Data.Description.DefaultValue},
		{t.GetDescription, t.GetTheme, button.Data.Theme.DefaultValue},
	}

	for _, stage := range stages {
		if currentVar == stage.getCurrent() {
			currentVar = stage.getNext()
			if stage.defaultValue == nil {
				return nextStageTicketButton(ctx, md, button, currentVar)
			}
		}
	}

	if currentVar == t.GetTheme() {
		// чистим данные
		err := md.chatState.ClearCacheOmitemptyFields(md.cacheDB, md.msg.UserID, md.msg.LineID)

		return SendAnswer(ctx, md, md.chatState.PreviousState, err)
	}

	return finalSend(ctx, md, "", errors.New("не найдено куда направить пользователя по кнопке Назад"))
}

// Проверить нажата ли BackButton
func getGoToIfClickedBackBtn(btn *botconfig_parser.Button, md *MultiData, ignoreHistoryBack bool) (goTo string) {
	if btn != nil && btn.BackButton {
		if !ignoreHistoryBack {
			_ = md.chatState.HistoryStateBack(md.cacheDB, md.msg.UserID, md.msg.LineID)
		}

		if md.chatState.PreviousState != database.GREETINGS {
			goTo = md.chatState.PreviousState
		} else {
			goTo = database.START
		}
	}
	return
}

// обработать событие произошедшее в чате
func processMessage(md *MultiData) (string, error) {
	time.Sleep(250 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var err error
	chatState, msg, bot, menu := md.chatState, md.msg, md.bot, md.menu

	switch msg.MessageType {
	// Первый запуск.
	case messages.MESSAGE_TREATMENT_START_BY_USER:
		switch chatState.CurrentState {
		case database.START,
			database.GREETINGS,
			database.FAIL_QNA:
			return chatState.CurrentState, nil
		}
		err := chatState.HistoryStateClear(md.cacheDB, msg.UserID, msg.LineID)
		return database.GREETINGS, err

	// Нажатие меню ИЛИ Любое сообщение (текст, файл, попытка звонка).
	case messages.MESSAGE_CALL_START_TREATMENT,
		messages.MESSAGE_CALL_START_NO_TREATMENT,
		messages.MESSAGE_TREATMENT_START_BY_SPEC,
		messages.MESSAGE_TREATMENT_CLOSE,
		messages.MESSAGE_TREATMENT_CLOSE_ACTIVE:
		err = bot.connect.Start(ctx, msg.UserID)
		_ = chatState.HistoryStateClear(md.cacheDB, msg.UserID, msg.LineID)
		return database.GREETINGS, err

	case messages.MESSAGE_NO_FREE_SPECIALISTS:
		err = bot.connect.RerouteTreatment(ctx, msg.UserID)
		_ = chatState.HistoryStateClear(md.cacheDB, msg.UserID, msg.LineID)
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
						return qnaResponse(ctx, md, currentMenu)
					}
				}
			}

			if menu.FirstGreeting {
				_ = bot.connect.Send(ctx, msg.UserID, menu.GreetingMessage, nil)
				time.Sleep(time.Second)
			}
			return SendAnswer(ctx, md, database.START, err)

		// пользователь попадет сюда в случае регистрации заявки
		case database.CREATE_TICKET:
			btn := GetClickedButton(menu, chatState.CurrentState, text)
			tBtn := chatState.GetCacheSavedButton().TicketButton
			ticket := database.Ticket{}

			// переходим если нажата Отмена
			goTo := getGoToIfClickedBackBtn(btn, md, true)
			if goTo != "" {
				// перейти в определенное меню если настроен параметр goto
				if tBtn.Goto != "" {
					goTo = tBtn.Goto
				}

				// чистим данные
				err := chatState.ClearCacheOmitemptyFields(md.cacheDB, msg.UserID, msg.LineID)
				if err != nil {
					return finalSend(ctx, md, "", err)
				}

				return SendAnswer(ctx, md, goTo, err)
			}

			// узнаем имя переменной
			varName, exist := chatState.GetCacheVar(database.VAR_FOR_SAVE)
			if !exist {
				return finalSend(ctx, md, "", err)
			}

			// проверяем нажата ли кнопка Назад
			if btn != nil && btn.Goto == database.CREATE_TICKET_PREV_STAGE {
				return prevStageTicketButton(ctx, md, tBtn, varName)
			}

			switch varName {
			case ticket.GetTheme(), ticket.GetDescription():
				textForSave := msg.Text
				// если кнопка перехода к следующему шагу
				if btn != nil && btn.Goto == database.CREATE_TICKET {
					textForSave = ""
				}
				err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, varName, database.TicketPart{Name: &textForSave})
				if err != nil {
					return finalSend(ctx, md, "", err)
				}

				switch varName {
				case ticket.GetTheme():
					return nextStageTicketButton(ctx, md, tBtn, ticket.GetDescription())
				case ticket.GetDescription():
					return nextStageTicketButton(ctx, md, tBtn, ticket.GetExecutor())
				}

			case ticket.GetExecutor(), ticket.GetService(), ticket.GetServiceType():
				// если кнопка перехода к следующему шагу
				if btn != nil && btn.Goto == database.CREATE_TICKET {
					err = bot.connect.Send(ctx, msg.UserID, menu.ErrorMessages.TicketButton.StepCannotBeSkipped, nil)
					return database.CREATE_TICKET, err
				} else {
					switch varName {
					case ticket.GetExecutor():
						// получаем список специалистов
						listSpecs, err := bot.connect.GetSpecialists(ctx, msg.LineID)
						if err != nil {
							return finalSend(ctx, md, "", err)
						}

						for _, v := range listSpecs {
							fio := strings.TrimSpace(fmt.Sprintf("%s %s %s", v.Surname, v.Name, v.Patronymic))
							if msg.Text == fio {
								err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, varName, database.TicketPart{ID: v.UserID, Name: &msg.Text})
								if err != nil {
									return finalSend(ctx, md, "", err)
								}

								return nextStageTicketButton(ctx, md, tBtn, ticket.GetService())
							}
						}

					case ticket.GetService():
						// получаем данные для заявок
						kinds, err := bot.connect.GetTicketDataKinds(ctx, nil, chatState.User.CounterpartOwnerID)
						if err != nil {
							return finalSend(ctx, md, "", err)
						}

						for _, v := range kinds {
							if msg.Text == v.Name {
								err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, varName, database.TicketPart{ID: v.ID, Name: &msg.Text})
								if err != nil {
									return finalSend(ctx, md, "", err)
								}

								return nextStageTicketButton(ctx, md, tBtn, ticket.GetServiceType())
							}
						}

					case ticket.GetServiceType():
						// получаем данные для заявок
						types, err := bot.connect.GetTicketDataTypesWhereKind(ctx, nil, chatState.User.CounterpartOwnerID, chatState.Ticket.Service.ID)
						if err != nil {
							return finalSend(ctx, md, "", err)
						}

						for _, v := range types {
							if msg.Text == v.Name {
								err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, varName, database.TicketPart{ID: v.ID, Name: &msg.Text})
								if err != nil {
									return finalSend(ctx, md, "", err)
								}

								return nextStageTicketButton(ctx, md, tBtn, ticket.GetFinal())
							}
						}
					}

					// если не найдено значение то ошибка
					err = bot.connect.Send(ctx, msg.UserID, menu.ErrorMessages.TicketButton.ReceivedIncorrectValue, nil)
					return database.CREATE_TICKET, err
				}

			// этап регистрации заявки
			case ticket.GetFinal():
				if btn != nil && btn.Goto == database.CREATE_TICKET {
					// удаляем клавиатуру
					err = bot.connect.DropKeyboard(ctx, msg.UserID)
					if err != nil {
						return finalSend(ctx, md, "", err)
					}

					_ = bot.connect.Send(ctx, msg.UserID, "Заявка регистрируется, ожидайте...", nil)

					// регистрируем заявку
					r, err := us.CreateTicket(ctx, md.soapcl, msg.UserID, msg.LineID, chatState.GetCacheTicket())
					if err != nil {
						return finalSend(ctx, md, "", err)
					}

					// даем время чтобы загрузилась заявка
					for range 10 {
						time.Sleep(4 * time.Second)

						_, err := bot.connect.GetTicket(ctx, uuid.MustParse(r["ServiceRequestID"]))
						if err == nil {
							break
						}
					}

					// чистим данные
					err = chatState.ClearCacheOmitemptyFields(md.cacheDB, msg.UserID, msg.LineID)

					return SendAnswer(ctx, md, tBtn.Goto, err)
				}
			}

			err = bot.connect.Send(ctx, msg.UserID, menu.ErrorMessages.TicketButton.ExpectedButtonPress, nil)
			return database.CREATE_TICKET, err

		// пользователь попадет сюда в случае перехода в режим ожидания сообщения
		case database.WAIT_SEND:
			state := cache.GetState(bot.connect, ctx, md.cacheDB, msg.UserID, msg.LineID)

			// записываем введенные данные в переменную
			varName, ok := chatState.GetCacheVar(database.VAR_FOR_SAVE)
			if ok && varName != "" {
				_ = chatState.ChangeCacheVars(md.cacheDB, msg.UserID, msg.LineID, varName, msg.Text)
			}

			// чистим необязательные поля
			err = chatState.ClearCacheOmitemptyFields(md.cacheDB, msg.UserID, msg.LineID)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			// переходим если нажата BackButton
			btn := GetClickedButton(menu, chatState.CurrentState, text)
			goTo := getGoToIfClickedBackBtn(btn, md, true)
			if goTo != "" {
				return SendAnswer(ctx, md, goTo, err)
			}

			// выполнить действие кнопки
			gt, err := triggerButton(ctx, md, state.SavedButton)
			_ = chatState.HistoryStateAppend(md.cacheDB, msg.UserID, msg.LineID, gt)
			return gt, err

		default:
			currentMenu := chatState.CurrentState

			// В редисе может остаться состояние которого, нет в конфиге.
			cm, ok := menu.Menu[currentMenu]
			if !ok {
				logger.Warning("неизвестное состояние: ", currentMenu)
				err = bot.connect.Send(ctx, msg.UserID, menu.ErrorMessages.CommandUnknown, menu.GenKeyboard(database.START))
				return database.GREETINGS, err
			}

			// определяем какая кнопка была нажата
			btn := GetClickedButton(menu, currentMenu, text)

			if btn != nil {
				gt, err := triggerButton(ctx, md, btn)
				_ = chatState.HistoryStateAppend(md.cacheDB, msg.UserID, msg.LineID, gt)
				return gt, err
			} else { // Произвольный текст
				if !cm.QnaDisable && menu.UseQNA.Enabled {
					return qnaResponse(ctx, md, currentMenu)
				}
				err = bot.connect.Send(ctx, msg.UserID, menu.ErrorMessages.CommandUnknown, menu.GenKeyboard(currentMenu))
				return chatState.CurrentState, err
			}
		}
	case messages.MESSAGE_TREATMENT_TO_BOT,
		messages.MESSAGE_LINE_REROUTING_OTHER_LINE:
	default:
		panic(fmt.Sprintf("unexpected messages.MessageType: %#v", msg.MessageType))
	}
	return database.GREETINGS, errors.New("i don't know what i should do")
}

// ищем ответ в базе знаний. если находим то отправляем ответ пользователю если не находим то fail_qna_menu
func qnaResponse(ctx context.Context, md *MultiData, currentMenu string) (string, error) {
	var err error

	// logger.Info("QNA", msg, chatState)
	qnaText, needClose, requestID, resultID := getMessageFromQNA(ctx, md)
	if qnaText != "" {
		// Была подсказка
		go md.bot.connect.QnaSelected(ctx, requestID, resultID)

		if needClose {
			err = md.bot.connect.Send(ctx, md.msg.UserID, qnaText, nil)
			_ = md.bot.connect.CloseTreatment(ctx, md.msg.UserID)
			return currentMenu, err
		}

		err = md.bot.connect.Send(ctx, md.msg.UserID, qnaText, md.menu.GenKeyboard(currentMenu))
		return currentMenu, err
	}

	return SendAnswer(ctx, md, database.FAIL_QNA, err)
}

// выполнить действие кнопки
func triggerButton(ctx context.Context, md *MultiData, btn *botconfig_parser.Button) (string, error) {
	if btn == nil {
		return finalSend(ctx, md, "", fmt.Errorf("Кнопка не передана в triggerButton"))
	}

	var err error
	chatState, msg, bot, menu, cnf := md.chatState, md.msg, md.bot, md.menu, md.cnf

	goTo := btn.Goto
	if gt := getGoToIfClickedBackBtn(btn, md, false); gt != "" {
		goTo = gt
	}

	// отображаем содержимое Chat
	err = SendAnswerMenu(ctx, md, btn.Chat, nil)
	if err != nil {
		return finalSend(ctx, md, "", err)
	}

	if btn.CloseButton {
		err = bot.connect.CloseTreatment(ctx, msg.UserID)
		return database.GREETINGS, err
	}
	if btn.RedirectButton {
		err = bot.connect.RerouteTreatment(ctx, msg.UserID)
		return database.GREETINGS, err
	}
	if btn.AppointSpecButton != nil && *btn.AppointSpecButton != uuid.Nil {
		// проверяем доступен ли специалист
		ok, err := bot.connect.GetSpecialistAvailable(ctx, *btn.AppointSpecButton)
		if err != nil || !ok {
			return finalSend(ctx, md, menu.ErrorMessages.AppointSpecButton.SelectedSpecNotAvailable, err)
		}

		// назначаем если свободен
		err = bot.connect.AppointSpec(ctx, msg.UserID, cnf.SpecID, *btn.AppointSpecButton)
		return database.GREETINGS, err
	}
	if btn.AppointRandomSpecFromListButton != nil && len(*btn.AppointRandomSpecFromListButton) != 0 {
		// получаем список свободных специалистов
		r, err := bot.connect.GetSpecialistsAvailable(ctx)
		if err != nil || len(r) == 0 {
			return finalSend(ctx, md, menu.ErrorMessages.AppointRandomSpecFromListButton.SpecsNotAvailable, err)
		}

		// создаем словарь id специалистов которых мы хотели бы назначить
		specIDs := make(map[uuid.UUID]struct{})
		for _, id := range *btn.AppointRandomSpecFromListButton {
			specIDs[id] = struct{}{}
		}

		// ищем среди свободных специалистов нужных
		neededSpec := make([]uuid.UUID, 0)
		for _, v := range r {
			if _, exists := specIDs[v]; exists {
				neededSpec = append(neededSpec, v)
			}
		}

		// проверяем есть ли хотя бы 1 свободный специалист
		lenNeededSpec := len(neededSpec)
		if lenNeededSpec == 0 {
			return finalSend(ctx, md, menu.ErrorMessages.AppointRandomSpecFromListButton.SpecsNotAvailable, err)
		}

		// назначаем случайного специалиста из списка
		seed := time.Now().UnixNano()
		rns := rand.NewSource(seed)
		rng := rand.New(rns)
		randomIndex := rng.Intn(lenNeededSpec)
		err = bot.connect.AppointSpec(ctx, msg.UserID, cnf.SpecID, neededSpec[randomIndex])
		return database.GREETINGS, err
	}
	if btn.RerouteButton != nil && *btn.RerouteButton != uuid.Nil {
		// проверяем доступна линия пользователю
		r, err := bot.connect.GetSubscriptions(ctx, msg.UserID, *btn.RerouteButton)
		if err != nil {
			return finalSend(ctx, md, "", err)
		}
		if len(r) == 0 {
			return finalSend(ctx, md, menu.ErrorMessages.RerouteButton.SelectedLineNotAvailable, err)
		}

		// назначаем если все ок
		err = bot.connect.Reroute(ctx, msg.UserID, *btn.RerouteButton, "")
		if err != nil {
			return finalSend(ctx, md, "", err)
		}
		return database.GREETINGS, err
	}
	if btn.ExecButton != "" {
		// удаляем пробелы после {{ и до }}
		for strings.Contains(btn.ExecButton, "{{ ") || strings.Contains(btn.ExecButton, " }}") {
			btn.ExecButton = strings.ReplaceAll(btn.ExecButton, "{{ ", "{{")
			btn.ExecButton = strings.ReplaceAll(btn.ExecButton, " }}", "}}")
		}

		// разбиваем шаблон на части (команда и аргументы) чтобы исключить возможность выйти за кавычки
		cmdParts, err := shellquote.Split(btn.ExecButton)
		if err != nil {
			return finalSend(ctx, md, "", err)
		}

		// заполняем каждую часть шаблона отдельно
		for k, part := range cmdParts {
			cmdParts[k], err = fillTemplateWithInfo(chatState, part)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}
		}

		// выполняем команду на устройстве
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		cmdOutput, err := cmd.CombinedOutput()
		if err != nil {
			return finalSend(ctx, md, "Ошибка: "+err.Error(), err)
		}

		// выводим результат и завершаем
		_ = bot.connect.Send(ctx, msg.UserID, string(cmdOutput), nil)
		goTo := database.FINAL
		if btn.Goto != "" {
			goTo = btn.Goto
		}
		return SendAnswer(ctx, md, goTo, err)
	}
	if btn.SaveToVar != nil {
		// настройка клавиатуры
		keyboard := &[][]requests.KeyboardKey{}
		for _, v := range btn.SaveToVar.OfferOptions {
			r, err := fillTemplateWithInfo(chatState, v)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}
			*keyboard = append(*keyboard, []requests.KeyboardKey{{Text: r}})
		}
		*keyboard = append(*keyboard, *menu.GenKeyboard(database.WAIT_SEND)...)

		// Сообщаем пользователю что требуем и запускаем ожидание данных
		if btn.SaveToVar.SendText != nil && *btn.SaveToVar.SendText != "" {
			r, err := fillTemplateWithInfo(chatState, *btn.SaveToVar.SendText)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}

			_ = bot.connect.Send(ctx, msg.UserID, r, keyboard)
		} else {
			// выводим default WAIT_SEND меню в случае отсутствия настроек текста
			err = SendAnswerMenu(ctx, md, menu.Menu[database.WAIT_SEND].Answer, keyboard)
			if err != nil {
				return finalSend(ctx, md, "", err)
			}
		}

		// сохраняем имя переменной куда будем записывать результат
		_ = chatState.ChangeCacheVars(md.cacheDB, msg.UserID, msg.LineID, database.VAR_FOR_SAVE, btn.SaveToVar.VarName)

		// сохраняем ссылку на кнопку которая будет выполнена после завершения
		if btn.SaveToVar.DoButton != nil {
			err = chatState.ChangeCacheSavedButton(md.cacheDB, msg.UserID, msg.LineID, btn.SaveToVar.DoButton)
		}

		return database.WAIT_SEND, err
	}
	if btn.TicketButton != nil {
		// сохраняем ссылку на кнопку которая была нажата
		err = chatState.ChangeCacheSavedButton(md.cacheDB, msg.UserID, msg.LineID, btn)
		if err != nil {
			return finalSend(ctx, md, "", err)
		}

		t := database.Ticket{}

		// сохраняем id канала поступления заявки
		err = chatState.ChangeCacheTicket(md.cacheDB, msg.UserID, msg.LineID, t.GetChannel(), database.TicketPart{ID: btn.TicketButton.ChannelID})
		if err != nil {
			return finalSend(ctx, md, "", err)
		}

		gt, err := nextStageTicketButton(ctx, md, btn.TicketButton, t.GetTheme())
		return gt, err
	}

	// Сообщения при переходе на новое меню.
	return SendAnswer(ctx, md, goTo, err)
}

// определить какая кнопка была нажата
func GetClickedButton(menu *botconfig_parser.Levels, currentMenu, text string) (btn *botconfig_parser.Button) {
	btn = menu.GetButton(currentMenu, text)
	if btn == nil {
		text = strings.ReplaceAll(text, "«", "\"")
		text = strings.ReplaceAll(text, "»", "\"")
		btn = menu.GetButton(currentMenu, text)
	}
	return
}

// выполнить Send и вывести Final меню
func finalSend(ctx context.Context, md *MultiData, finalMsg string, err error) (string, error) {
	if finalMsg == "" {
		finalMsg = md.menu.ErrorMessages.ButtonProcessing
	}
	_ = md.bot.connect.Send(ctx, md.msg.UserID, finalMsg, nil)

	// чистим данные чтобы избежать повторных ошибок
	_ = md.chatState.HistoryStateClear(md.cacheDB, md.msg.UserID, md.msg.LineID)

	return SendAnswer(ctx, md, database.FINAL, err)
}

// getMessageFromQNA - Метод возвращает ответ с Базы Знаний, и флаг, если это сообщение закрывает обращение.
func getMessageFromQNA(ctx context.Context, md *MultiData) (answer string, needClose bool, requestID uuid.UUID, resultID uuid.UUID) {
	qnaAnswer, err := md.bot.connect.GetQNA(ctx, md.msg.UserID, false, false)
	if err != nil {
		return
	}

	for _, v := range qnaAnswer.Answers {
		if v.Accuracy > 0 {
			resultID = v.ID
			needClose = v.AnswerSource == "GOODBYES"
			requestID = qnaAnswer.RequestID
			answer = v.Text
			return
		}
	}

	return
}
