package bot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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

		err = changeState(c, &msg, &chatState, newState)
		if err != nil {
			logger.Warning("Error changeState", err)
		}
	}(cCp, msg)

	c.Status(http.StatusOK)
}

func changeState(c *gin.Context, msg *messages.Message, chatState *database.Chat, toState string) error {
	cache := c.MustGet("cache").(*bigcache.BigCache)

	if chatState.CurrentState == toState {
		return nil
	}

	chatState.PreviousState = chatState.CurrentState
	chatState.CurrentState = toState

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

func getState(c *gin.Context, msg *messages.Message) database.Chat {
	cache := c.MustGet("cache").(*bigcache.BigCache)

	var chatState database.Chat

	dbStateKey := msg.UserId.String() + ":" + msg.LineId.String()

	b, err := cache.Get(dbStateKey)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			logger.Info("No state in cache for " + msg.UserId.String() + ":" + msg.LineId.String())
			chatState = database.Chat{
				PreviousState: database.GREETINGS,
				CurrentState:  database.GREETINGS,
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
			msg.Send(c, menu.Menu[goTo].Answer[i].Chat, toSend)
		}
		if menu.Menu[goTo].Answer[i].File != "" {
			if isImage, filePath, err := getFileInfo(menu.Menu[goTo].Answer[i].File, filesDir); err == nil {
				msg.SendFile(c, isImage, menu.Menu[goTo].Answer[i].File, filePath, &menu.Menu[goTo].Answer[i].FileText, toSend)
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func processMessage(c *gin.Context, msg *messages.Message, chatState *database.Chat) (string, error) {
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
			return database.START, nil
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

			btn := menu.GetButton(currentMenu, text)
			if btn == nil {
				text = strings.ReplaceAll(text, "«", "\"")
				text = strings.ReplaceAll(text, "»", "\"")
				btn = menu.GetButton(currentMenu, text)
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
						msg.Send(c, btn.Chat[i].Chat, nil)
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
						return finalSend(c, msg, menu, cnf.FilesDir, "Выбранный специалист недоступен", err)
					}
					err = msg.AppointSpec(c, *btn.AppointSpecButton)
					return database.GREETINGS, err
				}
				if btn.RerouteButton != nil && *btn.RerouteButton != uuid.Nil {
					r, err := msg.GetSubscriptions(c, *btn.RerouteButton)
					if err != nil {
						return finalSend(c, msg, menu, cnf.FilesDir, "", err)
					}
					if len(r) == 0 {
						return finalSend(c, msg, menu, cnf.FilesDir, "Выбранная линия недоступна", err)
					}

					err = msg.Reroute(c, *btn.RerouteButton, "")
					if err != nil {
						return finalSend(c, msg, menu, cnf.FilesDir, "", err)
					}
					return database.GREETINGS, err
				}
				if btn.ExecButton != "" {
					// получаем данные о пользователе
					userData, err := msg.GetSubscriber(c)
					if err != nil {
						return finalSend(c, msg, menu, cnf.FilesDir, "", err)
					}

					// формируем шаблон
					templ, err := template.New("cmd").Parse(btn.ExecButton)
					if err != nil {
						return finalSend(c, msg, menu, cnf.FilesDir, "", err)
					}

					// заполняем шаблон
					var templOutput bytes.Buffer
					err = templ.Execute(&templOutput, userData)
					if err != nil {
						return finalSend(c, msg, menu, cnf.FilesDir, "", err)
					}

					// выполняем команду на устройстве
					cmdParts := strings.Fields(templOutput.String())
					var cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
					cmdOutput, err := cmd.CombinedOutput()
					if err != nil {
						return finalSend(c, msg, menu, cnf.FilesDir, "Ошибка: "+err.Error(), err)
					}

					// выводим результат и завершаем
					msg.Send(c, string(cmdOutput), nil)
					goTo := database.FINAL
					SendAnswer(c, msg, menu, goTo, cnf.FilesDir)
					return goTo, err
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

func finalSend(c *gin.Context, msg *messages.Message, menu *botconfig_parser.Levels, filesDir, finalMsg string, err error) (string, error) {
	if finalMsg != "" {
		msg.Send(c, finalMsg, nil)
	} else {
		msg.Send(c, "Во время обработки вашего запроса произошла ошибка", nil)
	}
	goTo := database.FINAL
	SendAnswer(c, msg, menu, goTo, filesDir)
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
