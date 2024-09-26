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
	"slices"
	"time"

	"connect-text-bot/bot/client"
	"connect-text-bot/bot/requests"
	"connect-text-bot/config"
	"connect-text-bot/logger"

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
		LineId uuid.UUID `json:"line_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		UserId uuid.UUID `json:"user_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`

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
func (msg *Message) GetQNA(cnf *config.Conf, skip_greetings, skip_goodbyes bool) *AutofaqRequestBody {
	data := requests.Qna{
		LineID:        msg.LineId,
		UserId:        msg.UserId,
		SkipGreetings: skip_greetings,
		SkipGoodbyes:  skip_goodbyes,
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
func (msg *Message) QnaSelected(cnf *config.Conf, request_id, result_id uuid.UUID) {
	data := requests.Selected{
		Request_ID: request_id,
		Result_ID:  result_id,
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
		LineID: msg.LineId,
		UserId: msg.UserId,
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
		LineID:   msg.LineId,
		UserId:   msg.UserId,
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
		LineID: msg.LineId,
		UserId: msg.UserId,
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
		LineID: msg.LineId,
		UserId: msg.UserId,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/drop/treatment/", nil, "application/json", jsonData)

	return err
}

func (msg *Message) StartAndReroute(cnf *config.Conf) error {
	msg.Start(cnf)
	data := requests.TreatmentRequest{
		LineID: msg.LineId,
		UserId: msg.UserId,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/appoint/start/", nil, "application/json", jsonData)

	return err
}

// Попытаться назначить конкретного специалиста
func (msg *Message) AppointSpec(c *gin.Context, appoint_spec uuid.UUID) error {
	cnf := c.MustGet("cnf").(*config.Conf)
	data := requests.TreatmentWithSpecAndAuthorRequest{
		LineID:   msg.LineId,
		UserId:   msg.UserId,
		SpecId:   appoint_spec,
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
func (msg *Message) GetSpecialistAvailable(c *gin.Context, spec_id uuid.UUID) (available bool, err error) {
	spec_ids, err := msg.GetSpecialistsAvailable(c)
	if err != nil {
		return
	}

	return slices.Contains(spec_ids, spec_id), err
}

// Получить список специалистов доступных по линии
func (msg *Message) GetSpecialistsAvailable(c *gin.Context) (spec_ids []uuid.UUID, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)

	r, err := client.Invoke(cnf, http.MethodGet, "/line/specialists/"+msg.LineId.String()+"/available/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &spec_ids)
	return
}

// Перевод обращения на другую линию
func (msg *Message) Reroute(c *gin.Context, line_id uuid.UUID, quote string) error {
	cnf := c.MustGet("cnf").(*config.Conf)
	data := requests.TreatmentReroute{
		LineID:   msg.LineId,
		UserId:   msg.UserId,
		ToLineId: line_id,
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
	r, err := client.Invoke(cnf, http.MethodGet, "/line/subscriber/"+msg.UserId.String()+"/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение списка линий, подключенных пользователям
func (msg *Message) GetSubscriptions(c *gin.Context, line_id uuid.UUID) (content requests.Subscriptions, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)
	var v = url.Values{}
	v.Add("user_id", msg.UserId.String())
	v.Add("line_id", line_id.String())

	r, err := client.Invoke(cnf, http.MethodGet, "/line/subscriptions/", v, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение информации о специалисте
func (msg *Message) GetSpecialist(c *gin.Context, spec_id uuid.UUID) (content requests.User, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)
	r, err := client.Invoke(cnf, http.MethodGet, "/line/specialist/"+spec_id.String()+"/", nil, "application/json", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(r, &content)
	return
}

// Получение информации о специалистах
func (msg *Message) GetSpecialists(c *gin.Context, line_id uuid.UUID) (content requests.Users, err error) {
	cnf := c.MustGet("cnf").(*config.Conf)

	var v = url.Values{}
	if line_id != uuid.Nil {
		v.Add("line_id", line_id.String())
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
		LineID: msg.LineId,
		UserId: msg.UserId,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	_, err = client.Invoke(cnf, http.MethodPost, "/line/drop/keyboard/", nil, "application/json", jsonData)
	return
}

// Метод позволяет отправить файл или изображение в чат
func (msg *Message) SendFile(c *gin.Context, isImage bool, fileName string, filepath string, comment *string, keyboard *[][]requests.KeyboardKey) error {
	cnf := c.MustGet("cnf").(*config.Conf)

	data := requests.FileRequest{
		LineID:   msg.LineId,
		UserId:   msg.UserId,
		AuthorID: cnf.SpecID,
		FileName: fileName,
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

	file, err := os.Open(filepath)
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
