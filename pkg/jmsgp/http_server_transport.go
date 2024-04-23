package jmsgp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
)

const DefaultHTTPBodyMaxBytes = 1048576 // 1MB
const DefaultMessageIdHTTPHeader = "X-Request-Id"

func TargetFromHTTPRequestURLPath(r *http.Request) string {
	return path.Base(r.URL.Path) // XXX
}

func TargetFromHTTPRequestURLPathValue(name string) func(*http.Request) string {
	return func (r *http.Request) string {
		return r.PathValue(name)
	}
}

type HTTPServerTransport struct {
	BodyMaxBytes      int64
	MessageIdHeader   string
	ExtractTargetFunc func(*http.Request) string
	hub               *Hub
}

func NewHTTPServerTransport(hub *Hub) *HTTPServerTransport {
	return &HTTPServerTransport{
		BodyMaxBytes:      DefaultHTTPBodyMaxBytes,
		MessageIdHeader:   DefaultMessageIdHTTPHeader,
		ExtractTargetFunc: TargetFromHTTPRequestURLPath,
		hub:               hub,
	}
}

func (t *HTTPServerTransport) HandleRequest(w http.ResponseWriter, r *http.Request) error {
	env := &httpEnvelope{
		ctx:    r.Context(),
		peer:   &httpPeer{w: w},
		id:     r.Header.Get(t.MessageIdHeader),
		target: t.ExtractTargetFunc(r),
		body:   http.MaxBytesReader(w, r.Body, t.BodyMaxBytes),
	}

	if dispatchErr := t.dispatch(r.Context(), r, env); dispatchErr != nil {
		if sendErr := env.Respond(env.ctx, dispatchErr); sendErr != nil {
			return errors.Join(dispatchErr, sendErr)
		}
		return dispatchErr
	}
	return nil
}

func (t *HTTPServerTransport) dispatch(ctx context.Context, r *http.Request, env *httpEnvelope) error {
	mediatype, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediatype != "application/json" {
		return &jmsgpError{code: InvalidMessageErrCode, text: "Content-Type header is not application/json"}
	}

	return t.hub.Dispatch(ctx, env)
}

type httpPeer struct {
	w http.ResponseWriter
}

var _ Peer = (*httpPeer)(nil)

func (p *httpPeer) Send(ctx context.Context, target, id string, data any) error {
	return WriteHTTPResponse(ctx, p.w, target, id, data)
}

func WriteHTTPResponse(ctx context.Context, w http.ResponseWriter, target, id string, data any) error {
	msg := Message{
		Id:     id,
		Target: target,
	}

	dErr, ok := data.(error)
	if ok {
		msg.setError(dErr)
	} else {
		msg.Data = data
	}

	httpStatus := mapErrorCodeToHTTPStatus(msg.ErrorCode)
	body, marshalErr := json.Marshal(&msg)
	if marshalErr != nil {
		fallbackMsg := Message{
			Id:        id,
			Target:    target,
			ErrorCode: InternalErrCode,
		}
		httpStatus = mapErrorCodeToHTTPStatus(fallbackMsg.ErrorCode)
		fallbackBody, err := json.Marshal(&fallbackMsg)
		if err != nil {
			// Must not happen.
			panic(err)
		}
		body = fallbackBody
		marshalErr = fmt.Errorf("jmsgp send msg marshal: %w", marshalErr)

	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_, writeErr := w.Write(body)
	if writeErr != nil {
		writeErr = fmt.Errorf("jmsgp send msg write: %w", writeErr)
	}

	return errors.Join(marshalErr, writeErr)
}

func mapErrorCodeToHTTPStatus(code string) int {
	switch {
	case code == "":
		return http.StatusOK
	case code == InvalidTargetErrCode:
		return http.StatusNotFound
	case code == InvalidMessageErrCode:
		return http.StatusBadRequest
	case code == InvalidDataErrCode:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

type httpEnvelope struct {
	ctx    context.Context
	peer   Peer
	id     string
	target string
	body   io.ReadCloser
}

var _ Envelope = (*httpEnvelope)(nil)

func (e *httpEnvelope) Context() context.Context {
	return e.ctx
}

func (e *httpEnvelope) Peer() Peer {
	return e.peer
}

func (e *httpEnvelope) Id() string {
	return e.id
}

func (e *httpEnvelope) Target() string {
	return e.target
}

func (e *httpEnvelope) Respond(ctx context.Context, data any) error {
	return e.peer.Send(ctx, e.target, e.id, data)
}

func (e *httpEnvelope) BindData(ctx context.Context, dst any) error {
	dec := json.NewDecoder(e.body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &jmsgpError{code: InvalidMessageErrCode, text: msg}

		// https://github.com/golang/go/issues/25956
		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("request body contains badly-formed JSON")
			return &jmsgpError{code: InvalidMessageErrCode, text: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := "invalid message data"
			issue := fmt.Sprintf("invalid type, expected %q", unmarshalTypeError.Type.Name())
			return &jmsgpError{code: InvalidDataErrCode, text: msg, data: map[string]string{unmarshalTypeError.Field: issue}}

		// https://github.com/golang/go/issues/29035
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			msg := "invalid message data"
			issue := "unknown field"
			return &jmsgpError{code: InvalidDataErrCode, text: msg, data: map[string]string{fieldName: issue}}

		case errors.Is(err, io.EOF):
			msg := "request body must not be empty"
			return &jmsgpError{code: InvalidMessageErrCode, text: msg}

		case errors.As(err, &maxBytesError):
			msg := fmt.Sprintf("request body must not be larger than %d bytes", maxBytesError.Limit)
			return &jmsgpError{code: InvalidMessageErrCode, text: msg}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "request body must only contain a single JSON object"
		return &jmsgpError{code: InvalidMessageErrCode, text: msg}
	}

	validator, ok := dst.(Validator)
	if ok {
		if issues := validator.Validate(ctx); len(issues) > 0 {
			msg := "invalid message data"
			return &jmsgpError{code: InvalidDataErrCode, text: msg, data: issues}
		}
	}

	return nil
}
