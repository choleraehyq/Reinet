package reinet

import (
	"reflect"
	"html/template"
	"regexp"
	"net/http"
)

const (
	// DELETE HTTP method
	DELETE = "DELETE"
	// GET HTTP method
	GET = "GET"
	// HEAD HTTP method
	HEAD = "HEAD"
	// OPTIONS HTTP method
	OPTIONS = "OPTIONS"
	// PATCH HTTP method
	PATCH = "PATCH"
	// POST HTTP method
	POST = "POST"
	// PUT HTTP method
	PUT = "PUT"
	// TRACE HTTP method
	TRACE = "TRACE"
)

type Context struct {
	req *http.Request
	res http.ResponseWriter
	formParams map[string]string
	urlQueryParams map[string]string
}

type handler interface{}

type route struct {
	regex *regexp.Regexp
	params map[int]string
	handler handler
	method string
}

var mainServer *ReiServer
var Sessions *Manager

func init() {
	mainServer = NewServer()
	Sessions = nil
	initSession()
	Sessions, _ = NewManager("default", "reinetSessionID", 3600)
}

func wrap(handleFunc handler) handler { 
	return handleFunc
}

func Get(pattern string, handleFunc handler) {
	mainServer.addRoute(pattern, wrap(handleFunc), GET)
}

func Post(pattern string, handleFunc handler) {
	mainServer.addRoute(pattern, wrap(handleFunc), POST)
}

func Delete(pattern string, handleFunc handler) {
	mainServer.addRoute(pattern, wrap(handleFunc), DELETE)
}

func Put(pattern string, handleFunc handler) {
	mainServer.addRoute(pattern, wrap(handleFunc), PUT)
}

func Patch(pattern string, handleFunc handler) {
	mainServer.addRoute(pattern, wrap(handleFunc), PATCH)
}

func GivenMethod(pattern string, handleFunc handler, method string) {
	mainServer.addRoute(pattern, wrap(handleFunc), method)
}

func SetStatic(url string, path string) {
	(*(mainServer.staticDir))[url] = path
}

func Run(addr string) {
	go Sessions.GC()
	http.ListenAndServe(addr, mainServer)
}

func UseProvider(providerName string, provider Provider) {
	AddProvider(providerName, provider)
	Sessions.provider = provides[providerName]
}

func SetSessionExpires(expires int64) {
	Sessions.maxLifeTime = expires
}

func RenderTemplate(ctx Context, tmpl string, params interface{}) {
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		http.Error(ctx.res, err.Error(), http.StatusInternalServerError)
		return 
	}
	err := t.Execute(ctx.res, params)
	if err != nil {
		http.Error(ctx.res, err.Error(), http.StatusInternalServerError)
		return 
	}
}

func Redirect(ctx Context, redirectUrl string) {
	http.Redirect(ctx.res, ctx.req, redirectUrl, http.StatusFound)
}

func BeforeRequest(mid handler) {
	mainServer.addBefore(mid)
}

func afterRequest(mid handler) {
	mainServer.addAfter(mid)
}