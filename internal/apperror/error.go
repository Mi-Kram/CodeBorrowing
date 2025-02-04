package apperror

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type AppError struct {
	Err              error  `json:"-"`
	Message          string `json:"message,omitempty"`
	DeveloperMessage string `json:"-"`
	Code             string `json:"-"`
}

func NewAppError(message, code, developerMessage string) *AppError {
	return &AppError{
		Err:              fmt.Errorf(message),
		Message:          message,
		DeveloperMessage: developerMessage,
		Code:             code,
	}
}

func (e *AppError) Error() string {
	return e.Err.Error()
}

func (e *AppError) Marshal() []byte {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return bytes
}

func BadRequestError(message string) *AppError {
	return NewAppError(message, string(rune(http.StatusBadRequest)), "some thing wrong with user data")
}

func SystemError(developerMessage string) *AppError {
	return NewAppError("system error", string(rune(http.StatusInternalServerError)), developerMessage)
}

var ErrNotFound = NewAppError("Not Found", string(rune(http.StatusNotFound)), "")
