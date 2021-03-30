package gogearbox_testings

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/gogearbox/gearbox"
	jsoniter "github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
	"net/http"
	"net/http/httputil"
	"strings"
)

// handlerFunc defines the handler used by middleware as return value.
type handlerFunc func(ctx gearbox.Context)

// handlersChain defines a handlerFunc array.
type handlersChain []handlerFunc

// Context defines the current context of request and handlers/middlewares to execute
type fake_context struct {
	requestCtx  *fasthttp.RequestCtx
	paramValues map[string]string
	handlers    handlersChain
	index       int
}

type fake_request struct {
	request *http.Request
	context *fake_context
	body    []byte
}

func (fr *fake_request) SetParam(param, value string) {
	fr.context.paramValues[param] = value
}

func (fr *fake_request) SetHeader(key, value string) {
	fr.request.Header.Set(key, value)
}

func (fr *fake_request) Run(handler handlerFunc) (fasthttp.Response, error) {

	dumpRequest, err := httputil.DumpRequest(fr.request, true)
	if err != nil {
		return fasthttp.Response{}, err
	}
	err = fr.context.requestCtx.Request.Read(bufio.NewReader(bytes.NewReader(dumpRequest)))
	if err != nil {
		return fasthttp.Response{}, err
	}
	fr.context.requestCtx.Request.SetBody(fr.body)
	handler(fr.context)
	return fr.context.requestCtx.Response, nil
}

func NewFakeRequest(method string, url string, body []byte) (*fake_request, error) {
	req, e := http.NewRequest(method, url, bytes.NewBuffer(body))
	if e != nil {
		return nil, e
	}
	return &fake_request{
		request: req,
		context: &fake_context{
			requestCtx:  &fasthttp.RequestCtx{},
			paramValues: map[string]string{},
		},
		body: body,
	}, nil
}

// Next function is used to successfully pass from current middleware to next middleware.
// if the middleware thinks it's okay to pass it
func (ctx *fake_context) Next() {
	ctx.index++
	if ctx.index < len(ctx.handlers) {
		ctx.handlers[ctx.index](ctx)
	}
}

// Param returns value of path parameter specified by key
func (ctx *fake_context) Param(key string) string {
	return ctx.paramValues[key]
}

// Context returns Fasthttp context
func (ctx *fake_context) Context() *fasthttp.RequestCtx {
	return ctx.requestCtx
}

// SendBytes sets body of response for []byte type
func (ctx *fake_context) SendBytes(value []byte) gearbox.Context {
	ctx.requestCtx.Response.SetBodyRaw(value)
	return ctx
}

// SendString sets body of response for string type
func (ctx *fake_context) SendString(value string) gearbox.Context {
	ctx.requestCtx.SetBodyString(value)
	return ctx
}

// SendJSON converts any interface to json, sets it to the body of response
// and sets content type header to application/json.
func (ctx *fake_context) SendJSON(in interface{}) error {
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	raw, err := json.Marshal(in)
	// Check for errors
	if err != nil {
		return err
	}

	// Set http headers
	ctx.requestCtx.Response.Header.SetContentType(gearbox.MIMEApplicationJSON)
	ctx.requestCtx.Response.SetBodyRaw(raw)

	return nil
}

// Status sets the HTTP status code
func (ctx *fake_context) Status(status int) gearbox.Context {
	ctx.requestCtx.Response.SetStatusCode(status)
	return ctx
}

// Get returns the HTTP request header specified by field key
func (ctx *fake_context) Get(key string) string {
	return gearbox.GetString(ctx.requestCtx.Request.Header.Peek(key))
}

// Set sets the response's HTTP header field key to the specified key, value
func (ctx *fake_context) Set(key, value string) {
	ctx.requestCtx.Response.Header.Set(key, value)
}

// Query returns the query string parameter in the request url
func (ctx *fake_context) Query(key string) string {
	return gearbox.GetString(ctx.requestCtx.QueryArgs().Peek(key))
}

// Body returns the raw body submitted in a POST request
func (ctx *fake_context) Body() string {
	return gearbox.GetString(ctx.requestCtx.Request.Body())
}

// SetLocal stores value with key within request scope and it is accessible through
// handlers of that request
func (ctx *fake_context) SetLocal(key string, value interface{}) {
	ctx.requestCtx.SetUserValue(key, value)
}

// GetLocal gets value by key which are stored by SetLocal within request scope
func (ctx *fake_context) GetLocal(key string) interface{} {
	return ctx.requestCtx.UserValue(key)
}

// ParseBody parses request body into provided struct
// Supports decoding theses types: application/json
func (ctx *fake_context) ParseBody(out interface{}) error {
	contentType := gearbox.GetString(ctx.requestCtx.Request.Header.ContentType())
	if strings.HasPrefix(contentType, gearbox.MIMEApplicationJSON) {
		json := jsoniter.ConfigCompatibleWithStandardLibrary
		return json.Unmarshal(ctx.requestCtx.Request.Body(), out)
	}

	return fmt.Errorf("content type '%s' is not supported, "+
		"please open a request to support it "+
		"(https://github.com/gogearbox/gearbox/issues/new?template=feature_request.md)",
		contentType)
}
