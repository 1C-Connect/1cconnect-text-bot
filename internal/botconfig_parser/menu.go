package botconfig_parser

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Levels struct {
	Menu map[string]*Menu `yaml:"menus"`

	UseQNA QNA `yaml:"use_qna"`

	BackButton                      *Button `yaml:"back_button"`
	CloseButton                     *Button `yaml:"close_button"`
	RedirectButton                  *Button `yaml:"redirect_button"`
	AppointSpecButton               *Button `yaml:"appoint_spec_button"`
	AppointRandomSpecFromListButton *Button `yaml:"appoint_random_spec_from_list_button"`
	RerouteButton                   *Button `yaml:"reroute_button"`
	ExecButton                      *Button `yaml:"exec_button"`
	SaveToVar                       *Button `yaml:"save_to_var"`
	TicketButton                    *Button `yaml:"ticket_button"`

	GreetingMessage string `yaml:"greeting_message"`
	FirstGreeting   bool   `yaml:"first_greeting"`

	// сообщения об ошибках
	ErrorMessages ErrorMessages `yaml:"error_messages"`
}

type Menu struct {
	Answer []*Answer `yaml:"answer"`

	// вывести кнопки при отображение меню
	Buttons []*Buttons `yaml:"buttons,omitempty"`
	// выполнить действие по кнопке при отображение меню
	DoButton *Button `yaml:"do_button,omitempty"`

	QnaDisable bool `yaml:"qna_disable"`
}

type ErrorMessages struct {
	// Команда неизвестна. Попробуйте еще раз
	CommandUnknown string `yaml:"command_unknown"`
	// Во время обработки вашего запроса произошла ошибка
	ButtonProcessing string `yaml:"button_processing"`
	// Ошибка: Не удалось отправить файл
	FailedSendFile string `yaml:"failed_send_file"`

	AppointSpecButton struct {
		// Выбранный специалист недоступен
		SelectedSpecNotAvailable string `yaml:"selected_spec_not_available"`
	} `yaml:"appoint_spec_button"`

	AppointRandomSpecFromListButton struct {
		// Специалисты данной области недоступны
		SpecsNotAvailable string `yaml:"specs_not_available"`
	} `yaml:"appoint_random_spec_from_list_button"`

	RerouteButton struct {
		// Выбранная линия недоступна
		SelectedLineNotAvailable string `yaml:"selected_line_not_available"`
	} `yaml:"reroute_button"`

	TicketButton struct {
		// Данный этап нельзя пропустить
		StepCannotBeSkipped string `yaml:"step_cannot_be_skipped"`
		// Получено некорректное значение. Повторите попытку
		ReceivedIncorrectValue string `yaml:"received_incorrect_value"`
		// Ожидалось нажатие на кнопку. Повторите попытку
		ExpectedButtonPress string `yaml:"expected_button_press"`
	} `yaml:"ticket_button"`
}

type QNA struct {
	Enabled bool `yaml:"enabled"`
}

type Answer struct {
	// сообщение при переходе на меню
	Chat string `yaml:"chat"`
	// путь к файлу
	File string `yaml:"file,omitempty"`
	// сопроводительный текст к файлу
	FileText string `yaml:"file_text,omitempty"`
}

type Buttons struct {
	Button Button
}

type NestedMenu struct {
	ID      string     `yaml:"id"`
	Answer  []*Answer  `yaml:"answer"`
	Buttons []*Buttons `yaml:"buttons"`

	QnaDisable bool `yaml:"qna_disable"`
}

type Button struct {
	// id кнопки
	ButtonID string `yaml:"id"`
	// текст кнопки
	ButtonText string `yaml:"text"`
	// сообщение
	Chat []*Answer `yaml:"chat,omitempty"`
	// закрыть обращение
	CloseButton bool `yaml:"close_button,omitempty"`
	// перевести на специалиста
	RedirectButton bool `yaml:"redirect_button,omitempty"`
	// вернуться назад
	BackButton bool `yaml:"back_button,omitempty"`
	// перевести на специалиста по id
	AppointSpecButton *uuid.UUID `yaml:"appoint_spec_button,omitempty"`
	// перевести на случайного специалиста из списка id
	AppointRandomSpecFromListButton *[]uuid.UUID `yaml:"appoint_random_spec_from_list_button,omitempty"`
	// Перевод обращения на другую линию
	RerouteButton *uuid.UUID `yaml:"reroute_button,omitempty"`
	// Выполнить команду на стороне сервера
	ExecButton string `yaml:"exec_button,omitempty"`
	// получить и сохранить текст введенный пользователем
	SaveToVar *SaveToVar `yaml:"save_to_var,omitempty"`
	// зарегистрировать заявку
	TicketButton *TicketButton `yaml:"ticket_button,omitempty"`
	// перейти в меню
	Goto string `yaml:"goto"`
	// вложенное меню
	NestedMenu *NestedMenu `yaml:"menu"`
}

// добавить таб в начале каждой строки
func tabLines(input, tabs string) string {
	lines := strings.Split(input, "\n")
	for i := range lines {
		lines[i] = tabs + lines[i]
	}
	return strings.Join(lines, "\n")
}

func (b Menu) View() (btnStr string) {
	btnStr += fmt.Sprintf("\nlen(Answer): %d", len(b.Answer))
	btnStr += fmt.Sprintf("\nlen(Buttons): %d", len(b.Buttons))

	if b.DoButton != nil {
		btnStr += fmt.Sprintf("\nDoButton: {%s}", b.DoButton.View())
	} else {
		btnStr += "\nDoButton: nil"
	}

	btnStr += fmt.Sprintf("\nQnaDisable: %v", b.QnaDisable)

	return fmt.Sprintf("%s\n", tabLines(btnStr, "\t"))
}

func (b Button) View() (btnStr string) {
	btnStr += fmt.Sprintf("\nButtonID: %s", b.ButtonID)
	btnStr += fmt.Sprintf("\nButtonText: %s", b.ButtonText)
	btnStr += fmt.Sprintf("\nlen(Chat): %d", len(b.Chat))

	btnCnf := make([]string, 0)
	if b.CloseButton {
		btnCnf = append(btnCnf, "CloseButton")
	}
	if b.RedirectButton {
		btnCnf = append(btnCnf, "RedirectButton")
	}
	if b.BackButton {
		btnCnf = append(btnCnf, "BackButton")
	}
	if b.AppointSpecButton != nil {
		btnCnf = append(btnCnf, "AppointSpecButton")
	}
	if b.AppointRandomSpecFromListButton != nil {
		btnCnf = append(btnCnf, "AppointRandomSpecFromListButton")
	}
	if b.RerouteButton != nil {
		btnCnf = append(btnCnf, "RerouteButton")
	}
	if b.ExecButton != "" {
		btnCnf = append(btnCnf, "ExecButton")
	}
	if b.SaveToVar != nil {
		btnCnf = append(btnCnf, "SaveToVar")
	}
	if b.TicketButton != nil {
		btnCnf = append(btnCnf, "TicketButton")
	}

	btnStr += fmt.Sprintf("\nModifier: %v", btnCnf)

	btnStr += fmt.Sprintf("\nGoto: %s", b.Goto)
	if b.NestedMenu != nil {
		btnStr += fmt.Sprintf("\nNestedMenu ID: %v", b.NestedMenu.ID)
	}

	return fmt.Sprintf("%s\n", tabLines(btnStr, "\t"))
}

func (b SaveToVar) View() (btnStr string) {
	btnStr += fmt.Sprintf("\nVarName: %s", b.VarName)
	btnStr += fmt.Sprintf("\nSendText: %s", *b.SendText)
	btnStr += fmt.Sprintf("\nlen(OfferOptions): %d", len(b.OfferOptions))

	if b.DoButton != nil {
		btnStr += fmt.Sprintf("\nDoButton: {%s}", b.DoButton.View())
	} else {
		btnStr += "\nDoButton: nil"
	}

	return fmt.Sprintf("%s\n", tabLines(btnStr, "\t"))
}

func (b TicketButton) View() (btnStr string) {
	btnStr += fmt.Sprintf("\nChannelID: %s", b.ChannelID)
	btnStr += fmt.Sprintf("\nlen(TicketInfo): %d", len([]rune(b.TicketInfo)))

	formatPartTicket := func(fieldName string, pt *PartTicket) string {
		if pt == nil {
			return fmt.Sprintf("\n%s: nil", fieldName)
		}
		defaultValue := "nil"
		if pt.DefaultValue != nil {
			defaultValue = *pt.DefaultValue
		}
		return fmt.Sprintf("\n%s: { Text: %s, DefaultValue: %s }", fieldName, pt.Text, defaultValue)
	}

	btnStr += "\nData: {"
	btnStr += tabLines(formatPartTicket("Theme", b.Data.Theme), "\t")
	btnStr += tabLines(formatPartTicket("Description", b.Data.Description), "\t")
	btnStr += tabLines(formatPartTicket("Executor", b.Data.Executor), "\t")
	btnStr += tabLines(formatPartTicket("Service", b.Data.Service), "\t")
	btnStr += tabLines(formatPartTicket("ServiceType", b.Data.ServiceType), "\t")
	btnStr += "\n}"

	btnStr += fmt.Sprintf("\nGoto: %s", b.Goto)

	return fmt.Sprintf("%s\n", tabLines(btnStr, "\t"))
}

type TicketButton struct {
	// Канал связи
	ChannelID uuid.UUID `yaml:"channel_id"`
	// шаблон текста, где выводятся заполненные данные заявки
	TicketInfo string `yaml:"ticket_info"`
	// данные заполняемой заявки
	Data *struct {
		// тема заявки
		Theme *PartTicket `yaml:"theme"`
		// описание заявки
		Description *PartTicket `yaml:"description"`
		// исполнитель
		Executor *PartTicket `yaml:"executor"`
		// услуга
		Service *PartTicket `yaml:"service"`
		// тип услуги
		ServiceType *PartTicket `yaml:"type"`
	} `yaml:"data"`

	// перейти в меню при окончание или отмене
	Goto string `yaml:"goto"`
}

type PartTicket struct {
	// не показывать кнопку пропуска
	Required bool `yaml:"required,omitempty"`
	// текст приглашения к вводу
	Text string `yaml:"text,omitempty"`
	// значение по умолчанию
	DefaultValue *string `yaml:"value,omitempty"`
}

type SaveToVar struct {
	// имя переменной в которую будет сохранено сообщение пользователя
	VarName string `yaml:"var_name"`
	// сообщение при нажатие на кнопку
	SendText *string `yaml:"send_text,omitempty"`

	// список вариантов из которых пользователь может выбрать ответ
	OfferOptions []string `yaml:"offer_options,omitempty"`
	// после получения сообщения пользователя выполнить действие по кнопке
	DoButton *Button `yaml:"do_button"`
}

// применить настройки "по умолчанию" для кнопки
func (b *Button) SetDefault(default_ Button) {
	if b.ButtonID == "" {
		b.ButtonID = default_.ButtonID
	}
	if b.ButtonText == "" {
		b.ButtonText = default_.ButtonText
	}
	if len(b.Chat) == 0 {
		b.Chat = default_.Chat
	}
}
