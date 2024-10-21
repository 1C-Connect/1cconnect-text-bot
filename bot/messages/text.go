package messages

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"time"

	"connect-text-bot/bot/client"
	"connect-text-bot/bot/requests"
	"connect-text-bot/internal/config"
	"connect-text-bot/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MessageType int

const (
	MESSAGE_TEXT                      MessageType = 1
	MESSAGE_CALL_START_TREATMENT      MessageType = 20
	MESSAGE_CALL_START_NO_TREATMENT   MessageType = 21
	MESSAGE_FILE                      MessageType = 70
	MESSAGE_TREATMENT_START_BY_USER   MessageType = 80
	MESSAGE_TREATMENT_START_BY_SPEC   MessageType = 81
	MESSAGE_TREATMENT_CLOSE           MessageType = 82
	MESSAGE_NO_FREE_SPECIALISTS       MessageType = 83
	MESSAGE_LINE_REROUTING_OTHER_LINE MessageType = 89
	MESSAGE_TREATMENT_CLOSE_ACTIVE    MessageType = 90
	MESSAGE_TREATMENT_TO_BOT          MessageType = 200
)

type (
	Message struct {
		LineID uuid.UUID `json:"line_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		UserID uuid.UUID `json:"user_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`

		MessageID     uuid.UUID   `json:"message_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		MessageType   MessageType `json:"message_type" binding:"required" example:"1"`
		MessageAuthor *uuid.UUID  `json:"author_id" binding:"omitempty" example:"4e48509f-6366-4897-9544-46f006e47074"`
		MessageTime   string      `json:"message_time" binding:"required" example:"1"`
		Text          string      `json:"text" example:"Привет"`
		Data          struct {
			Redirect string `json:"redirect"`
		} `json:"data"`
	}

	AutofaqAnswer struct {
		ID           uuid.UUID `json:"id"`
		Text         string    `json:"text"`
		Accuracy     float32   `json:"accuracy"`
		AnswerSource string    `json:"answer_source"`
	}

	AutofaqRequestBody struct {
		RequestID uuid.UUID       `json:"request_id"`
		Question  string          `json:"question"`
		Answers   []AutofaqAnswer `json:"answers"`
	}
)

// GetQNA - Метод позволяет получить варианты ответов на вопрос пользователя в сервисе AutoFAQ.
func (msg *Message) GetQNA(cnf *config.Conf, skipGreetings, skipGoodbyes bool) *AutofaqRequestBody {
	data := requests.Qna{
		LineID:        msg.LineID,
		UserID:        msg.UserID,
		SkipGreetings: skipGreetings,
		SkipGoodbyes:  skipGoodbyes,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Warning("text - GetQNA", err)
	}

	body, err := client.Invoke(cnf, http.MethodPost, "/line/qna/", nil, "application/json", jsonData)
	if err != nil {
		logger.Warning("text - GetQNA", err)
	}

	resp := &AutofaqRequestBody{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logger.Warning("text - GetQNA", err)
	}

	// Debug
	logger.Debug("text - GetQNA", resp)

	return resp
}

// Отметить выбранный вариант подсказки
func (_ *Message) QnaSelected(cnf *config.Conf, requestID, resultID uuid.UUID) {
	data := requests.Selected{
		RequestID: requestID,
		ResultID:  resultID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Warning("text - GetQNA", err)
	}

	body, err := client.Invoke(cnf, http.MethodPut, "/line/qna/selected/", nil, "application/json", jsonData)
	if err != nil {
		logger.Warning("text - QnaSelected", err, body)
	}
}

func (msg *Message) Start(cnf *config.Conf) error {
	data := requests.DropKeyboardRequest{
		LineID: msg.LineID,
		UserID: msg.UserID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/drop/keyboard/", nil, "application/json", jsonData)

	return err
}

// Отправить сообщение в чат
func (msg *Message) Send(c *gin.Context, text string, keyboard *[][]requests.KeyboardKey) error {
	cnf := c.MustGet("cnf").(*config.Conf)

	data := requests.MessageRequest{
		LineID:   msg.LineID,
		UserID:   msg.UserID,
		AuthorID: cnf.SpecID,
		Text:     text,
		Keyboard: keyboard,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/send/message/", nil, "application/json", jsonData)

	return err
}

func (msg *Message) RerouteTreatment(c *gin.Context) error {
	cnf := c.MustGet("cnf").(*config.Conf)

	data := requests.TreatmentRequest{
		LineID: msg.LineID,
		UserID: msg.UserID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/appoint/start/", nil, "application/json", jsonData)

	return err
}

// Закрыть текущее обращение
func (msg *Message) CloseTreatment(c *gin.Context) error {
	cnf := c.MustGet("cnf").(*config.Conf)

	time.Sleep(500 * time.Millisecond)

	data := requests.TreatmentRequest{
		LineID: msg.LineID,
		UserID: msg.UserID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/drop/treatment/", nil, "application/json", jsonData)

	return err
}

func (msg *Message) StartAndReroute(cnf *config.Conf) error {
	_ = msg.Start(cnf)
	data := requests.TreatmentRequest{
		LineID: msg.LineID,
		UserID: msg.UserID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/appoint/start/", nil, "application/json", jsonData)

	return err
}

// Попытаться назначить конкретного специалиста
func (msg *Message) AppointSpec(c *gin.Context, appointSpec uuid.UUID) error {
	cnf := c.MustGet("cnf").(*config.Conf)
	data := requests.TreatmentWithSpecAndAuthorRequest{
		LineID:   msg.LineID,
		UserID:   msg.UserID,
		SpecID:   appointSpec,
		AuthorID: cnf.SpecID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/appoint/spec/", nil, "application/json", jsonData)

	return err
}

// Проверить доступен ли специалист по id
func (msg *Message) GetSpecialistAvailable(c *gin.Context, specID uuid.UUID) (available bool, err error) {
	specIDs, err := msg.GetSpecialistsAvailable(c)
	if err != nil {
		return
	}

	return slices.Contains(specIDs, specID), err
}

// Получить список специалистов доступных по линии
func (msg *Message) GetSpecialistsAvailable(c *gin.Context) (specIDs []uuid.UUID, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)

	r, err := client.Invoke(cnf, http.MethodGet, "/line/specialists/"+msg.LineID.String()+"/available/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &specIDs)
	return
}

// Перевод обращения на другую линию
func (msg *Message) Reroute(c *gin.Context, lineID uuid.UUID, quote string) error {
	cnf := c.MustGet("cnf").(*config.Conf)
	data := requests.TreatmentReroute{
		LineID:   msg.LineID,
		UserID:   msg.UserID,
		ToLineID: lineID,
		Quote:    quote,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/reroute/", nil, "application/json", jsonData)

	return err
}

// Получение информации о пользователе
func (msg *Message) GetSubscriber(c *gin.Context) (content requests.User, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)
	r, err := client.Invoke(cnf, http.MethodGet, "/line/subscriber/"+msg.UserID.String()+"/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение списка линий, подключенных пользователям
func (msg *Message) GetSubscriptions(c *gin.Context, lineID uuid.UUID) (content requests.Subscriptions, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)
	var v = url.Values{}
	v.Add("user_id", msg.UserID.String())
	v.Add("line_id", lineID.String())

	r, err := client.Invoke(cnf, http.MethodGet, "/line/subscriptions/", v, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение информации о специалисте
func (_ *Message) GetSpecialist(c *gin.Context, specID uuid.UUID) (content requests.User, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)
	r, err := client.Invoke(cnf, http.MethodGet, "/line/specialist/"+specID.String()+"/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение информации о специалистах
func (_ *Message) GetSpecialists(c *gin.Context, lineID uuid.UUID) (content requests.Users, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)

	var v = url.Values{}
	if lineID != uuid.Nil {
		v.Add("line_id", lineID.String())
	}

	r, err := client.Invoke(cnf, http.MethodGet, "/line/specialists/", v, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Скрыть цифровое меню
func (msg *Message) DropKeyboard(c *gin.Context) (err error) {
	cnf := c.MustGet("cnf").(*config.Conf)
	data := requests.TreatmentRequest{
		LineID: msg.LineID,
		UserID: msg.UserID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/drop/keyboard/", nil, "application/json", jsonData)
	return
}

// получить имя файла если оно имеет формат путь/имя_файла
func getFileName(fileName string) string {
	return filepath.Base(fileName)
}

// Метод позволяет отправить файл или изображение в чат
func (msg *Message) SendFile(c *gin.Context, isImage bool, fileName string, filePath string, comment *string, keyboard *[][]requests.KeyboardKey) error {
	cnf := c.MustGet("cnf").(*config.Conf)

	data := requests.FileRequest{
		LineID:   msg.LineID,
		UserID:   msg.UserID,
		AuthorID: cnf.SpecID,
		FileName: getFileName(fileName),
		Comment:  comment,
		Keyboard: keyboard,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	metaPartHeader := textproto.MIMEHeader{}
	metaPartHeader.Set("Content-Disposition", `form-data; name="meta"`)
	metaPartHeader.Set("Content-Type", "application/json")
	metaPart, err := writer.CreatePart(metaPartHeader)
	if err != nil {
		return err
	}
	_, _ = metaPart.Write(jsonData)

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	filePart, err := writer.CreateFormFile("file", fi.Name())
	if err != nil {
		return err
	}

	_, err = io.Copy(filePart, file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	if isImage {
		_, err = client.Invoke(cnf, http.MethodPost, "/line/send/image/", nil, writer.FormDataContentType(), body.Bytes())
		return err
	}
	_, err = client.Invoke(cnf, http.MethodPost, "/line/send/file/", nil, writer.FormDataContentType(), body.Bytes())
	return err
}
