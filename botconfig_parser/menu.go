package botconfig_parser

import "github.com/google/uuid"

type Levels struct {
	Menu map[string]*Menu `yaml:"menus"`

	UseQNA QNA `yaml:"use_qna"`

	BackButton                      *Button    `yaml:"back_button"`
	CloseButton                     *Button    `yaml:"close_button"`
	RedirectButton                  *Button    `yaml:"redirect_button"`
	AppointSpecButton               *Button    `yaml:"appoint_spec_button"`
	AppointRandomSpecFromListButton *Button    `yaml:"appoint_random_spec_from_list_button"`
	RerouteButton                   *Button    `yaml:"reroute_button"`
	ExecButton                      *Button    `yaml:"exec_button"`
	SaveToVar                       *SaveToVar `yaml:"save_to_var"`

	ErrorMessage    string `yaml:"error_message"`
	GreetingMessage string `yaml:"greeting_message"`
	FirstGreeting   bool   `yaml:"first_greeting"`
}

type Menu struct {
	Answer  []*Answer  `yaml:"answer"`
	Buttons []*Buttons `yaml:"buttons"`

	QnaDisable bool `yaml:"qna_disable"`
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
	// текст приглашения к вводу
	Text string `yaml:"text"`
	// значение по умолчанию
	DefaultValue string `yaml:"value,omitempty"`
}

type SaveToVar struct {
	// имя переменной в которую будет сохранено сообщение пользователя
	VarName string `yaml:"var_name"`
	// сообщение при нажатие на кнопку
	SendText *string `yaml:"send_text,omitempty"`

	// после получения сообщения пользователя выполнить действие по кнопке
	DoButton *Button `yaml:"do_button,omitempty"`
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
