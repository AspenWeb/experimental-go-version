package goaspen

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bitbucket.org/ww/goautoneg"
)

var (
	http406 = newErrHttp406()
	//DefaultCharsetDynamic = "utf-8"

	defaultAcceptHeader = "text/html,application/xhtml+xml," +
		"application/xml;q=0.9,*/*;q=0.8"
)

type errorHttp406 struct {
	msg string
}

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

func newErrHttp406() *errorHttp406 {
	return &errorHttp406{
		msg: "406: No acceptable media type available",
	}
}

func (me *errorHttp406) Error() string {
	return me.msg
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
	if len(contentType) == 0 {
		debugf("Ignoring call to `SetContentType` because argument is empty!")
		return
	}

	if strings.HasPrefix(contentType, "text/") && !strings.Contains(contentType, "charset=") {
		contentType = contentType + "; charset=utf-8" // XXX get default charset from config?
	}

	me.contentType = contentType
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

func (me *HTTPResponseWrapper) respond500(err error) {
	me.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isDebug {
		me.w.Header().Set("X-GoAspen-Error", fmt.Sprintf("%v", err))
	}
	me.w.WriteHeader(http.StatusInternalServerError)
	me.w.Write(http500Response)
}

func (me *HTTPResponseWrapper) respond406(err error) {
	me.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isDebug {
		me.w.Header().Set("X-GoAspen-Error", fmt.Sprintf("%v", err))
	}

	me.w.WriteHeader(http.StatusNotAcceptable)
	me.w.Write(http406Response)
}

func (me *HTTPResponseWrapper) Respond() {
	if me.err != nil {
		if _, ok := me.err.(*errorHttp406); ok {
			me.respond406(me.err)
			return
		}

		me.respond500(me.err)
		return
	}

	me.w.Header().Set("Content-Type", me.contentType)
	me.w.WriteHeader(me.statusCode)
	me.w.Write(me.bodyBytes)
}

func (me *HTTPResponseWrapper) RespondJSON() {
	if me.bodyObj == nil {
		me.respond500(errors.New("JSON response body not set!"))
		return
	}

	jsonBody, err := json.Marshal(me.bodyObj)
	if err != nil {
		me.respond500(err)
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
	accept := me.req.Header.Get(internalAcceptHeader)
	if len(accept) == 0 {
		accept = me.req.Header.Get(http.CanonicalHeaderKey("Accept"))
	}

	debugf("Looking up handler for Accept: %q", accept)
	debugf("Available content type handlers: %v", me.handledContentTypes)

	negotiated := goautoneg.Negotiate(accept, me.handledContentTypes)
	if len(negotiated) == 0 {
		me.err = http406
		return
	}

	handlerFunc, ok := me.contentTypeHandlers[negotiated]
	if ok {
		debugf("Calling handler %v for negotiated content type %q", handlerFunc, negotiated)
		handlerFunc(me)
	}
}
