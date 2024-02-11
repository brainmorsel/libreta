package jmsgp_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brainmorsel/libreta/pkg/jmsgp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testReq struct {
	Param string `json:"param"`
}

var _ jmsgp.Validator = (*testReq)(nil)

func (r *testReq) Validate(ctx context.Context) map[string]string {
	if !(r.Param == "value1" || r.Param == "value2") {
		return map[string]string{"param": "must be value1 or value2"}
	}
	return nil
}

type testResp struct {
	Result string `json:"result"`
}

func testHandler(env jmsgp.Envelope) error {
	var (
		req  testReq
		resp testResp
	)
	if err := env.BindData(env.Context(), &req); err != nil {
		return err
	}
	resp.Result = req.Param

	return env.Peer().Send(env.Context(), env.Target(), env.Id(), resp)
}

func testHandlerRPC(ctx context.Context, r *testReq) (*testResp, error) {
	return &testResp{Result: r.Param}, nil
}

type testCase struct {
	id          string
	urlPath     string
	reqBody     string
	contentType string

	wantBody   string
	wantStatus int
}

func (tc testCase) Id(id string) testCase          { tc.id = id; return tc }
func (tc testCase) Path(path string) testCase      { tc.urlPath = path; return tc }
func (tc testCase) ReqBody(body string) testCase   { tc.reqBody = body; return tc }
func (tc testCase) ContentType(ct string) testCase { tc.contentType = ct; return tc }
func (tc testCase) WantBody(body string) testCase  { tc.wantBody = body; return tc }
func (tc testCase) WantStatus(n int) testCase      { tc.wantStatus = n; return tc }

func TestHTTPServerTransport(t *testing.T) {
	hub := jmsgp.NewHub()
	hub.AddHandler("test-target", testHandler)
	hub.AddHandler("test-rpc", jmsgp.RPCHandler(testHandlerRPC))
	transport := jmsgp.NewHTTPServerTransport(hub)
	transport.BodyMaxBytes = 1024

	baseTC := testCase{
		id:          "test-id",
		urlPath:     "/api/test-target",
		reqBody:     `{"param": "value1"}`,
		contentType: "application/json",
		wantBody:    `{"id":"test-id","trg":"test-target","dat":{"result":"value1"}}`,
		wantStatus:  http.StatusOK,
	}
	tests := map[string]testCase{
		"success": baseTC,
		"success rpc": baseTC.
			Path("/api/test-rpc").
			WantBody(`{"id":"test-id","trg":"test-rpc","dat":{"result":"value1"}}`),
		"invalid target": baseTC.
			Path("/api/not-valid-target").
			WantStatus(http.StatusNotFound).
			WantBody(`{"id":"test-id","trg":"not-valid-target","err":"jmsgp.invalid_target","txt":"target not found"}`),
		"empty body": baseTC.
			ReqBody("").
			WantStatus(http.StatusBadRequest).
			WantBody(`{"id":"test-id","trg":"test-target","err":"jmsgp.invalid_message","txt":"request body must not be empty"}`),
		"large body": baseTC.
			ReqBody(`{"param":"` + strings.Repeat("X", 1024) + `"}`).
			WantStatus(http.StatusBadRequest).
			WantBody(`{"id":"test-id","trg":"test-target","err":"jmsgp.invalid_message","txt":"request body must not be larger than 1024 bytes"}`),
		"invalid content type": baseTC.
			ContentType("text/plain").
			WantStatus(http.StatusBadRequest).
			WantBody(`{"id":"test-id","trg":"test-target","err":"jmsgp.invalid_message","txt":"Content-Type header is not application/json"}`),
		"invalid json": baseTC.
			ReqBody(`{not-a-json}`).
			WantStatus(http.StatusBadRequest).
			WantBody(`{"id":"test-id","trg":"test-target","err":"jmsgp.invalid_message","txt":"request body contains badly-formed JSON (at position 2)"}`),
		"partial json": baseTC.
			ReqBody(`{"param":"value"`).
			WantStatus(http.StatusBadRequest).
			WantBody(`{"id":"test-id","trg":"test-target","err":"jmsgp.invalid_message","txt":"request body contains badly-formed JSON"}`),
		"multiple json objects": baseTC.
			ReqBody(`{"param":"value1"}{"param":"value2"}`).
			WantStatus(http.StatusBadRequest).
			WantBody(`{"id":"test-id","trg":"test-target","err":"jmsgp.invalid_message","txt":"request body must only contain a single JSON object"}`),
		"data type error": baseTC.
			ReqBody(`{"param": false}`).
			WantStatus(http.StatusBadRequest).
			WantBody(`{"id":"test-id","trg":"test-target","err":"jmsgp.invalid_data","txt":"invalid message data","dat":{"param":"invalid type, expected \"string\""}}`),
		"unexpected data field": baseTC.
			ReqBody(`{"unexpected": "inquisition"}`).
			WantStatus(http.StatusBadRequest).
			WantBody(`{"id":"test-id","trg":"test-target","err":"jmsgp.invalid_data","txt":"invalid message data","dat":{"unexpected":"unknown field"}}`),
		"invalid data": baseTC.
			ReqBody(`{"param": "INVALID"}`).
			WantStatus(http.StatusBadRequest).
			WantBody(`{"id":"test-id","trg":"test-target","err":"jmsgp.invalid_data","txt":"invalid message data","dat":{"param":"must be value1 or value2"}}`),
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.urlPath, bytes.NewReader([]byte(tc.reqBody)))
			req.Header.Set("Content-Type", tc.contentType)
			req.Header.Set(jmsgp.DefaultMessageIdHTTPHeader, tc.id)

			w := httptest.NewRecorder()
			transport.HandleRequest(w, req)
			res := w.Result()
			defer res.Body.Close()
			data, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.wantBody, string(data))
			assert.Equal(t, tc.wantStatus, res.StatusCode)
		})
	}
}
