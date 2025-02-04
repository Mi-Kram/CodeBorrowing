package apperror

import (
	"errors"
	"net/http"
)

type appHandler func(http.ResponseWriter, *http.Request) error

func Middleware(handler appHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err == nil {
			return
		}

		var appErr *AppError
		ok := errors.As(err, &appErr)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(SystemError(err.Error()).Marshal())
			return
		}

		statusCode := http.StatusBadRequest
		if errors.Is(appErr, ErrNotFound) {
			statusCode = http.StatusNotFound
		}

		w.WriteHeader(statusCode)
		_, _ = w.Write(appErr.Marshal())
	}
}
