package api

import "github.com/brainmorsel/libreta/pkg/jmsgp"

type InternalError struct {
	error
}

func (err InternalError) JMSGPError() (string, string) {
	return jmsgp.InternalErrCode, err.Error()
}

type InvalidContentType string

func (err InvalidContentType) Error() string {
	return string(err)
}

func (err InvalidContentType) JMSGPError() (string, string) {
	return "libreta.invalid_content_type", string(err)
}

var _ jmsgp.JMSGPError = InvalidContentType("")
