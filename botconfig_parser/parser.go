package botconfig_parser

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"connect-text-bot/bot/requests"
	"connect-text-bot/database"
	"connect-text-bot/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var lock = &sync.RWMutex{}
var levels *Levels

func InitLevels(path string) *Levels {
	if levels == nil {
		lock.Lock()
		defer lock.Unlock()
		if levels == nil {
			var err error
			levels, err = loadMenus(path)
			if err != nil {
				logger.Crit(err)
			}
			err = levels.checkMenus()
			if err != nil {
				logger.Crit(err)
			}
		} else {
			logger.Warning("Levels already created")
		}
	} else {
		logger.Warning("Levels already created")
	}
	return levels
}

func (l *Levels) UpdateLevels(path string) error {
	newLevel, err := loadMenus(path)
	if err != nil {
		return err
	}
	if err := newLevel.checkMenus(); err != nil {
		return err
	}
	*levels = *newLevel
	return nil
}

func loadMenus(path string) (*Levels, error) {
	input, _ := ioutil.ReadFile(path)

	menu := &Levels{}

	if err := yaml.Unmarshal(input, &menu); err != nil {
		return nil, err
	}
	if err := nestedToFlat(menu, &Levels{Menu: CopyMap(menu.Menu)}); err != nil {
		return nil, err
	}

	return menu, menu.checkMenus()
}

func defaultFinalMenu() *Menu {
	return &Menu{
		Answer: []*Answer{
			{Chat: "Могу ли я вам чем-то еще помочь?"},
		},
		Buttons: []*Buttons{
			{Button{ButtonID: "1", ButtonText: "Да", Goto: database.START}},
			{Button{ButtonID: "2", ButtonText: "Нет", Chat: []*Answer{{Chat: "Спасибо за обращение!"}}, CloseButton: true}},
			{Button{ButtonID: "0", ButtonText: "Соединить со специалистом", RedirectButton: true}},
		},
	}
}

func defaultFailQnaMenu() *Menu {
	return &Menu{
		Answer: []*Answer{
			{Chat: "Я Вас не понимаю.\n\nПопробуете еще раз или перевести обращение на специалиста?"},
		},
		Buttons: []*Buttons{
			{Button{ButtonID: "1", ButtonText: "Продолжить", BackButton: true}},
			{Button{ButtonID: "2", ButtonText: "Закрыть обращение", Chat: []*Answer{{Chat: "Спасибо за обращение!"}}, CloseButton: true}},
			{Button{ButtonID: "0", ButtonText: "Соединить со специалистом", RedirectButton: true}},
		},
	}
}

func CopyMap(m map[string]*Menu) map[string]*Menu {
	cp := make(map[string]*Menu)
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func nestedToFlat(main *Levels, nested *Levels) (err error) {
	for k, v := range nested.Menu {
		for _, b := range v.Buttons {
			if b.Button.NestedMenu != nil {
				if b.Button.NestedMenu.ID == "" {
					return fmt.Errorf("у вложенного уровня отсутсвует id %#v", b.Button)
				}
				if _, ok := main.Menu[b.Button.NestedMenu.ID]; !ok && b.Button.NestedMenu.ID != "" {
					menu := &Menu{
						Answer:  b.Button.NestedMenu.Answer,
						Buttons: b.Button.NestedMenu.Buttons,

						QnaDisable: b.Button.NestedMenu.QnaDisable,
					}

					main.Menu[b.Button.NestedMenu.ID] = menu
					b.Button.Goto = b.Button.NestedMenu.ID

					err = nestedToFlat(main, &Levels{Menu: map[string]*Menu{b.Button.NestedMenu.ID: menu}})
					if err != nil {
						return
					}
				} else {
					return fmt.Errorf("duplicate keys: %s", b.Button.NestedMenu.ID)
				}
			}
			if b.Button.NestedMenu == nil && !b.Button.BackButton && k != database.FINAL && b.Button.Goto == "" {
				b.Button.Goto = database.FINAL
				continue
			}
		}
	}
	return
}

func (l *Levels) checkMenus() error {
	if _, ok := l.Menu[database.START]; !ok {
		return fmt.Errorf("отсутствует меню %s", database.START)
	}
	if _, ok := l.Menu[database.FINAL]; !ok {
		l.Menu[database.FINAL] = defaultFinalMenu()
	}
	if l.UseQNA.Enabled {
		if _, ok := l.Menu[database.FAIL_QNA]; !ok {
			l.Menu[database.FAIL_QNA] = defaultFailQnaMenu()
		}
	}
	if l.ErrorMessage == "" {
		l.ErrorMessage = "Команда неизвестна. Попробуйте еще раз"
	}
	if l.GreetingMessage == "" {
		l.GreetingMessage = "Здравствуйте."
	}

	for k, v := range l.Menu {
		if len(v.Buttons) == 0 {
			return fmt.Errorf("отсутствуют кнопки: %s %#v", k, v)
		}
		for _, b := range v.Buttons {
			// Если кнопка CLOSE | REDIRECT | ... | BACK то применяем к ней дефолтные настройки
			var modifycatorCount = 0
			if b.Button.CloseButton && l.CloseButton != nil {
				b.Button.SetDefault(*l.CloseButton)
				modifycatorCount++
			}
			if b.Button.RedirectButton && l.RedirectButton != nil {
				b.Button.SetDefault(*l.RedirectButton)
				modifycatorCount++
			}
			if b.Button.BackButton && l.BackButton != nil {
				b.Button.SetDefault(*l.BackButton)
				modifycatorCount++
			}
			if b.Button.AppointSpecButton != nil && *b.Button.AppointSpecButton != uuid.Nil && l.AppointSpecButton != nil {
				b.Button.SetDefault(*l.AppointSpecButton)
				modifycatorCount++
			}
			if b.Button.RerouteButton != nil && *b.Button.RerouteButton != uuid.Nil && l.RerouteButton != nil {
				b.Button.SetDefault(*l.RerouteButton)
				modifycatorCount++
			}
			if b.Button.ExecButton != "" && l.ExecButton != nil {
				b.Button.SetDefault(*l.ExecButton)
			}

			if modifycatorCount > 1 {
				return fmt.Errorf("кнопка может иметь только один модификатор: %s %#v", k, b)
			}
			if b.Button.Goto != "" && b.Button.BackButton {
				return fmt.Errorf("back_button не может иметь goto: %s %#v", k, b)
			}
			if _, ok := l.Menu[b.Button.Goto]; !ok && !b.Button.BackButton && !b.Button.CloseButton && !b.Button.RedirectButton {
				return fmt.Errorf("кнопка ведет на несуществующий уровень: %s %#v", k, b)
			}
			if len(v.Answer) == 0 || !IsAnyAnswer(v.Answer) {
				return fmt.Errorf("отсутствует сообщение сопровождающее меню: %s", k)
			}
			if b.Button.ButtonText == "" {
				return fmt.Errorf("текст у кнопки не может быть пустой %s %#v", k, b)
			}
		}
	}
	return nil
}

func IsAnyAnswer(answer []*Answer) bool {
	countText, countFile := 0, 0
	for _, v := range answer {
		if v.Chat != "" {
			countText++
		}
		if v.File != "" {
			countFile++
		}
	}
	return countText != 0 || countFile != 0
}

func Quotes(s string) string {
	r := []rune(s)
	count := 0
	answer := strings.Builder{}
	for i, v := range r {
		// 34("). Ascii
		if v == 34 {
			if count%2 == 0 {
				answer.WriteString("«")
			} else {
				answer.WriteString("»")
			}
			count++
		} else {
			answer.WriteString(string(r[i]))
		}
	}
	return answer.String()
}

// GenKeyboard - создать клавиатуру
func (l *Levels) GenKeyboard(menu string) *[][]requests.KeyboardKey {
	answer := &[][]requests.KeyboardKey{}
	for _, v := range l.Menu[menu].Buttons {
		*answer = append(*answer, []requests.KeyboardKey{{Id: v.Button.ButtonID, Text: Quotes(v.Button.ButtonText)}})
	}
	if len(*answer) == 0 {
		return nil
	}
	return answer
}

func (l *Levels) GetButton(menu, text string) *Button {
	for _, v := range l.Menu[menu].Buttons {
		if text == strings.ToLower(strings.TrimSpace(v.Button.ButtonText)) || text == v.Button.ButtonID {
			return &v.Button
		}
	}
	return nil
}

// InjectLevels - Adds a levels to the Gin context
func InjectLevels(key string, levels *Levels) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(key, levels)
	}
}
