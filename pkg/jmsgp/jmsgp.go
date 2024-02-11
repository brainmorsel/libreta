// JSON Message Protocol.
package jmsgp

import (
	"context"
	"errors"
	"fmt"
)

// Protocol specific error codes. Application code must use namespaced errors, e.g. `app.some_error`.
const (
	InvalidTargetErrCode  = "jmsgp.invalid_target"
	InvalidMessageErrCode = "jmsgp.invalid_message"
	InvalidDataErrCode    = "jmsgp.invalid_data"
	InternalErrCode       = "jmsgp.internal"
)

type Message struct {
	Id        string `json:"id"`
	Target    string `json:"trg,omitempty"`
	ErrorCode string `json:"err,omitempty"`
	ErrorText string `json:"txt,omitempty"`
	Data      any    `json:"dat,omitempty"`
}

func (msg *Message) setError(err error) {
	var pErr JMSGPError
	switch {
	case errors.As(err, &pErr):
		code, text := pErr.JMSGPError()
		msg.ErrorCode = code
		msg.ErrorText = text
	default:
		// Don't set error description to prevent unintentional data leaks.
		msg.ErrorCode = InternalErrCode
	}
	dErr, ok := err.(JMSGPErrorData)
	if ok {
		msg.Data = dErr.JMSGPErrorData()
	}
}

type Envelope interface {
	Id() string
	Target() string
	Context() context.Context
	Peer() Peer
	BindData(ctx context.Context, dst any) error
	Respond(ctx context.Context, data any) error
}

type Peer interface {
	Send(ctx context.Context, target, id string, data any) error
}

type Validator interface {
	Validate(ctx context.Context) (issues map[string]string)
}

type JMSGPError interface {
	error
	JMSGPError() (code string, text string)
}

type JMSGPErrorData interface {
	JMSGPErrorData() (data any)
}

type jmsgpError struct {
	code string
	text string
	data any
}

var _ JMSGPError = (*jmsgpError)(nil)

func (err *jmsgpError) JMSGPError() (string, string) {
	return err.code, err.text
}

func (err *jmsgpError) JMSGPErrorData() any {
	return err.data
}

func (err *jmsgpError) Error() string {
	if err.text == "" {
		return err.code
	} else {
		return fmt.Sprintf("%s (%s)", err.code, err.text)
	}
}

type HandleFunc func(Envelope) error

func NewHub() *Hub {
	return &Hub{
		handlers: make(map[string]HandleFunc),
	}
}

// Hub dispatches message to appropriate handlers. Use NewHub() to instantiate.
type Hub struct {
	handlers map[string]HandleFunc
}

// AddHandler adds message handling function to dispatcher. Not thread safe at current time.
func (h *Hub) AddHandler(target string, f HandleFunc) {
	h.handlers[target] = f
}

func (h *Hub) Dispatch(ctx context.Context, env Envelope) error {
	f, ok := h.handlers[env.Target()]
	if !ok {
		return &jmsgpError{code: InvalidTargetErrCode, text: "target not found"}
	}
	return f(env)
}

// RPCHandler converts any function with compatible signature to RPC-like message handler.
func RPCHandler[I, O any](f func(context.Context, I) (O, error)) HandleFunc {
	return func(env Envelope) error {
		var (
			req  I
			resp O
		)
		if err := env.BindData(env.Context(), &req); err != nil {
			return err
		}
		resp, err := f(env.Context(), req)
		if err != nil {
			return err
		}

		return env.Respond(env.Context(), resp)
	}
}
