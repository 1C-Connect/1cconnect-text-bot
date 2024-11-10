package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"

	"connect-text-bot/internal/connect/requests"

	"github.com/google/uuid"
)

func (c Client) Start(ctx context.Context, userID uuid.UUID) error {
	data := requests.DropKeyboardRequest{
		LineID: c.lineID,
		UserID: userID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.Invoke(ctx, http.MethodPost, "/line/drop/keyboard/", nil, "application/json", jsonData)

	return err
}

// Скрыть цифровое меню
func (c Client) DropKeyboard(ctx context.Context, userID uuid.UUID) (err error) {
	data := requests.TreatmentRequest{
		LineID: c.lineID,
		UserID: userID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	_, err = c.Invoke(ctx, http.MethodPost, "/line/drop/keyboard/", nil, "application/json", jsonData)
	return
}

// Отправить сообщение в чат
func (c Client) Send(ctx context.Context, userID uuid.UUID, authorID *uuid.UUID, text string, keyboard *[][]requests.KeyboardKey) error {
	data := requests.MessageRequest{
		LineID:   c.lineID,
		UserID:   userID,
		AuthorID: authorID,
		Text:     text,
		Keyboard: keyboard,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.Invoke(ctx, http.MethodPost, "/line/send/message/", nil, "application/json", jsonData)

	return err
}

// Метод позволяет отправить файл или изображение в чат
func (c Client) SendFile(ctx context.Context, userID uuid.UUID, authorID *uuid.UUID, isImage bool, fileName string, filePath string, comment *string, keyboard *[][]requests.KeyboardKey) error {
	data := requests.FileRequest{
		LineID:   c.lineID,
		UserID:   userID,
		AuthorID: authorID,
		FileName: filepath.Base(fileName),
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
		_, err = c.Invoke(ctx, http.MethodPost, "/line/send/image/", nil, writer.FormDataContentType(), body.Bytes())
		return err
	}
	_, err = c.Invoke(ctx, http.MethodPost, "/line/send/file/", nil, writer.FormDataContentType(), body.Bytes())
	return err
}
