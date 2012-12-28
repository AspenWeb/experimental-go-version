package goaspen

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"bitbucket.org/ww/goautoneg"
)

var (
	defaultAcceptHeader = "text/html,application/xhtml+xml," +
		"application/xml;q=0.9,*/*;q=0.8"
)

type HTTPResponseWrapper struct {
	w   http.ResponseWriter
	req *http.Request

	statusCode int
	bodyBytes  []byte
	bodyObj    interface{}

	contentType         string
	contentTypeHandlers map[string]func(*HTTPResponseWrapper)
	handledContentTypes []string

	err error
}

func NewHTTPResponseWrapper(w http.ResponseWriter, req *http.Request) *HTTPResponseWrapper {
	return &HTTPResponseWrapper{
		w:   w,
		req: req,

		statusCode: http.StatusOK,
		bodyBytes:  []byte(""),

		contentType:         "text/html",
		contentTypeHandlers: make(map[string]func(*HTTPResponseWrapper)),

		err: nil,
	}
}

func (me *HTTPResponseWrapper) SetContentType(contentType string) {
	if len(contentType) > 0 {
		me.contentType = contentType
	}
}

func (me *HTTPResponseWrapper) SetBodyBytes(body []byte) {
	me.bodyBytes = body
}

func (me *HTTPResponseWrapper) SetBody(o interface{}) {
	me.bodyObj = o
}

func (me *HTTPResponseWrapper) SetStatusCode(sc int) {
	me.statusCode = sc
}

func (me *HTTPResponseWrapper) SetError(err error) {
	me.err = err
}

func (me *HTTPResponseWrapper) Respond500(err error) {
	me.w.Header().Set("Content-Type", "text/html")
	if isDebug {
		me.w.Header().Set("X-GoAspen-Error", fmt.Sprintf("%v", err))
	}
	me.w.WriteHeader(http.StatusInternalServerError)
	me.w.Write(http500Response)
}

func (me *HTTPResponseWrapper) Respond() {
	if me.err != nil {
		me.Respond500(me.err)
		return
	}

	me.w.Header().Set("Content-Type", me.contentType)
	me.w.WriteHeader(me.statusCode)
	me.w.Write(me.bodyBytes)
}

func (me *HTTPResponseWrapper) RespondJSON() {
	if me.bodyObj == nil {
		me.Respond500(errors.New("JSON response body not set!"))
		return
	}

	jsonBody, err := json.Marshal(me.bodyObj)
	if err != nil {
		me.Respond500(err)
		return
	}

	me.w.Header().Set("Content-Type", "application/json")
	me.w.WriteHeader(me.statusCode)
	me.w.Write(jsonBody)
}

func (me *HTTPResponseWrapper) RegisterContentTypeHandler(contentType string,
	handlerFunc func(*HTTPResponseWrapper)) {

	me.contentTypeHandlers[contentType] = handlerFunc
	me.handledContentTypes = append(me.handledContentTypes, contentType)
}

func (me *HTTPResponseWrapper) NegotiateAndCallHandler() {
	accept := me.req.Header.Get(http.CanonicalHeaderKey("Accept"))

	if len(accept) < 1 {
		accept = defaultAcceptHeader
	}

	negotiated := goautoneg.Negotiate(accept, me.handledContentTypes)
	handlerFunc, ok := me.contentTypeHandlers[negotiated]
	if ok {
		handlerFunc(me)
	}
}
