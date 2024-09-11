package botconfig_parser

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"sync"

	"connect-text-bot/bot/requests"
	"connect-text-bot/database"
	"connect-text-bot/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/goccy/go-yaml"
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

func loadMenus(pathCnf string) (*Levels, error) {
	input, _ := ioutil.ReadFile(pathCnf)
	dec := yaml.NewDecoder(bytes.NewBuffer(input), yaml.ReferenceDirs(path.Dir(pathCnf)), yaml.RecursiveDir(true))
	menu := &Levels{}
	if err := dec.Decode(menu); err != nil {
		return nil, err
	}

	// устанавливаем недостающие настройки
	for k, v := range CopyMap(menu.Menu) {
		err := nestedToFlat(menu, v.Buttons, k, 1)
		if err != nil {
			return nil, err
		}
	}

	// проверяем все меню
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

func defaultWaitSendMenu() *Menu {
	return &Menu{
		Answer: []*Answer{
			{Chat: "Введите ваше значение"},
		},
		Buttons: []*Buttons{
			{Button{ButtonID: "1", ButtonText: "Отмена", BackButton: true}},
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

func nestedToFlat(main *Levels, buttons []*Buttons, k string, depthLevel int) error {
	for _, b := range buttons {
		// добавляем goto на Final если меню не имеет продолжение
		if b.Button.SaveToVar == nil && b.Button.NestedMenu == nil && !b.Button.BackButton && k != database.FINAL && b.Button.Goto == "" {
			b.Button.Goto = database.FINAL
		}

		if b.Button.NestedMenu != nil {
			if b.Button.NestedMenu.Buttons != nil {
				err := nestedToFlat(main, b.Button.NestedMenu.Buttons, k, depthLevel+1)
				if err != nil {
					return err
				}
			}

			if b.Button.NestedMenu.ID == "" {
				return fmt.Errorf("отсутствует id у вложенного меню: %s %#v lvl:%d", k, b, depthLevel)
			}
			if _, exist := main.Menu[b.Button.NestedMenu.ID]; exist && b.Button.NestedMenu.ID != "" {
				return fmt.Errorf("уже существует меню с данным id(%s): %s %#v lvl:%d", b.Button.NestedMenu.ID, k, b, depthLevel)
			}
			menu := &Menu{
				Answer:     b.Button.NestedMenu.Answer,
				Buttons:    b.Button.NestedMenu.Buttons,
				QnaDisable: b.Button.NestedMenu.QnaDisable,
			}
			main.Menu[b.Button.NestedMenu.ID] = menu
			b.Button.Goto = b.Button.NestedMenu.ID
		}

		if b.Button.SaveToVar != nil && b.Button.SaveToVar.DoButton != nil {
			// задаем дефолтные настройки если кнопка не имеет дочерних кнопок
			btn := b.Button.SaveToVar.DoButton
			if btn.SaveToVar == nil {
				// задаем заглушку чтобы пропустить ошибку "текст у кнопки не может быть пустой"
				// используем ее тк она используется только для checkButton
				if btn.ButtonText == "" {
					b.Button.SaveToVar.DoButton.ButtonText = "<do_button>"
				}
			}

			if btn.NestedMenu != nil {
				b.Button.SaveToVar.DoButton.Goto = btn.NestedMenu.ID
			}

			err := nestedToFlat(main, []*Buttons{{Button: *b.Button.SaveToVar.DoButton}}, k, depthLevel+1)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Levels) checkMenus() error {
	if _, ok := l.Menu[database.START]; !ok {
		return fmt.Errorf("отсутствует меню %s", database.START)
	}
	if _, ok := l.Menu[database.FINAL]; !ok {
		l.Menu[database.FINAL] = defaultFinalMenu()
	}
	if _, ok := l.Menu[database.WAIT_SEND]; !ok {
		l.Menu[database.WAIT_SEND] = defaultWaitSendMenu()
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

	// проверка меню и подуровней
	for k, v := range l.Menu {
		if len(v.Buttons) == 0 {
			return fmt.Errorf("отсутствуют кнопки: %s %#v", k, v)
		}

		err := l.checkMenuLevels(v.Buttons, k, v, 1)
		if err != nil {
			return err
		}
	}
	return nil
}

// рекурсивная проверка меню и подуровней
func (l *Levels) checkMenuLevels(buttons []*Buttons, k string, v *Menu, depthLevel int) error {
	for _, b := range buttons {
		err := l.checkButton(b, k, v, depthLevel)
		if err != nil {
			return err
		}

		if b.Button.NestedMenu != nil && b.Button.NestedMenu.Buttons != nil {
			err := l.checkMenuLevels(b.Button.NestedMenu.Buttons, k, v, depthLevel+1)
			if err != nil {
				return err
			}
		}

		if b.Button.SaveToVar != nil && b.Button.SaveToVar.DoButton != nil {
			err := l.checkMenuLevels([]*Buttons{{Button: *b.Button.SaveToVar.DoButton}}, k, v, depthLevel+1)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// проверка кнопки на валидность
func (l *Levels) checkButton(b *Buttons, k string, v *Menu, depthLevel int) error {
	// Если кнопка CLOSE | REDIRECT | ... | BACK то применяем к ней дефолтные настройки
	var modifycatorCount = 0

	if b.Button.SaveToVar != nil {
		if b.Button.SaveToVar.VarName == "" {
			return fmt.Errorf("отсутствует var_name (имя переменной для сохранения данных): %s %#v lvl:%d", k, b, depthLevel)
		}
		if b.Button.SaveToVar.VarName == database.VAR_FOR_SAVE {
			return fmt.Errorf("используется зарезервированное имя переменной %s %#v lvl:%d", k, b, depthLevel)
		}
		if b.Button.SaveToVar.DoButton == nil {
			return fmt.Errorf("отсутствует do_button (действие которое выполнится после ответа пользователя): %s %#v lvl:%d", k, b, depthLevel)
		}
		if b.Button.SaveToVar.DoButton.BackButton {
			return fmt.Errorf("в do_button нельзя использовать back_button: %s %#v lvl:%d", k, b, depthLevel)
		}
		modifycatorCount++
	}

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
	if b.Button.AppointRandomSpecFromListButton != nil && len(*b.Button.AppointRandomSpecFromListButton) != 0 && l.AppointRandomSpecFromListButton != nil {
		b.Button.SetDefault(*l.AppointRandomSpecFromListButton)
		modifycatorCount++
	}
	if b.Button.RerouteButton != nil && *b.Button.RerouteButton != uuid.Nil && l.RerouteButton != nil {
		b.Button.SetDefault(*l.RerouteButton)
		modifycatorCount++
	}
	if b.Button.ExecButton != "" && l.ExecButton != nil {
		b.Button.SetDefault(*l.ExecButton)
		modifycatorCount++
	}

	if modifycatorCount > 1 {
		return fmt.Errorf("кнопка может иметь только один модификатор: %s %#v lvl:%d", k, b, depthLevel)
	}
	if b.Button.Goto != "" && b.Button.BackButton {
		return fmt.Errorf("back_button не может иметь goto: %s %#v lvl:%d", k, b, depthLevel)
	}
	if _, ok := l.Menu[b.Button.Goto]; b.Button.Goto != "" && !ok {
		return fmt.Errorf("кнопка ведет на несуществующий уровень: %s %#v lvl:%d", k, b, depthLevel)
	}
	if len(v.Answer) == 0 || !IsAnyAnswer(v.Answer) {
		return fmt.Errorf("отсутствует сообщение сопровождающее меню: %s lvl:%d", k, depthLevel)
	}
	if b.Button.SaveToVar == nil && b.Button.ButtonText == "" {
		return fmt.Errorf("текст у кнопки не может быть пустой %s %#v lvl:%d", k, b, depthLevel)
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
