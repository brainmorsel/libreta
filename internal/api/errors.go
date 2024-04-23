package api

import (
	"fmt"

	"github.com/brainmorsel/libreta/pkg/jmsgp"
)

type Error struct {
	Code string
	Msg  string
}

func (err *Error) Error() string {
	if err.Msg == "" {
		return fmt.Sprintf("api.Error(%s)", err.Code)
	} else {
		return fmt.Sprintf("api.Error(%s): %s", err.Code, err.Msg)
	}
}

func (err *Error) JMSGPError() (string, string) {
	return err.Code, err.Msg
}

var _ jmsgp.JMSGPError = (*Error)(nil)

func ErrInternal(err error) error {
	e := &Error{Code: jmsgp.InternalErrCode}
	if err != nil {
		e.Msg = err.Error()
	}
	return e
}

func ErrInvalidContentType(msg string) error {
	return &Error{Code: "libreta.invalid_content_type", Msg: msg}
}

func ErrNotFound() error {
	return &Error{Code: "libreta.not_found"}
}
