package reinet

import (
	"strconv"
	"strings"
	"log"
	"reflect"
	"regexp"
	"net/http"
	"os"
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

type ReiServer struct {
	routes []route
	logger *log.Logger
	staticDir *map[string]string
}

var mainServer *ReiServer
var Sessions *Manager

func init() {
	mainServer = NewServer()
	initSession()
	Sessions, _ = NewManager("default", "reinetSessionID", 3600)
	go Sessions.GC()
}

func NewServer() *ReiServer {
	m := make(map[string]string)
	return &ReiServer {
		logger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
		staticDir: &m,
	}
}

func (self *ReiServer) addRoute(pattern string, handleFunc handler, method string) {
	parts := strings.Split(pattern, "/")
	j := 0
	params := make(map[int]string)
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			expr := "([^/]+)"
			if index := strings.Index(part, "("); index != -1 {
				expr = part[index:]
				part = part[:index]
			}
			params[j] = part
			parts[i] = expr
			j++
		}
	}
	
	pattern = strings.Join(parts, "/")
	regex, regexErr := regexp.Compile(pattern)
	if regexErr != nil {
		panic(regexErr)
		return
	}
	
	newRoute := route {
		regex: regex, 
		params: params,
		handler: handleFunc,
		method: method,
	}
	self.routes = append(self.routes, newRoute)
}

func requireContext(handlerType reflect.Type) bool {
	if handlerType.NumIn() == 0 {
		return false
	}
	if firstParam := handlerType.In(0); firstParam.Kind() != reflect.Ptr || firstParam.Elem() != reflect.TypeOf(Context{}) {
		return false
	}
	return true
}

func (self *ReiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var started bool
	requestPath := r.URL.Path
	for prefix, staticDir := range *(self.staticDir) {
		if strings.HasPrefix(requestPath, prefix) {
			file := staticDir + requestPath[len(prefix):]
			http.ServeFile(w, r, file)
			started = true
			return
		}	
	}
	
	for _, route := range self.routes {
		if !route.regex.MatchString(requestPath) && route.method != r.Method {
			continue
		}
		
		matches := route.regex.FindStringSubmatch(requestPath)
		
		if len(matches[0]) != len(requestPath) {
			continue
		}
		
		r.ParseForm()
		formParams := make(map[string]string)
		if len(r.Form) > 0 {
			for k, v := range r.Form {
				formParams[k] = v[0]
			}
		}
		
		values := r.URL.Query()
		urlQueryParams := make(map[string]string)
		if len(values) > 0 {
			for k, v := range values {
				urlQueryParams[k] = v[0]
			}
		}
		
		params := make([]reflect.Value, 0)
		
		if requireContext(reflect.TypeOf(route.handler)) {
			ctx := &Context {
				req: r,
				res: w,
				formParams: formParams,
				urlQueryParams: urlQueryParams,
			}
			params = append(params, reflect.ValueOf(ctx))
		}
		
		for _, match := range matches[1:] {
			params = append(params, reflect.ValueOf(match))
		}
		
		rets := reflect.ValueOf(route.handler).Call(params)
		var content []byte
		if len(rets) != 0 {
			sval := rets[0]
			if sval.Kind() == reflect.String {
				content = []byte(sval.String())
			} else if sval.Kind() == reflect.Slice && sval.Type().Elem().Kind() == reflect.Uint8 {
				content = sval.Interface().([]byte)
			}
			w.Header().Set("Content-Type", strconv.Itoa(len(content)))
			_, err := w.Write(content)
			if err != nil {
				self.logger.Printf("Write content to client error: %v", err)
			}
		}
		started = true
		break
	}
	
	if started == false {
		http.NotFound(w, r)
	}
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
	http.ListenAndServe(addr, mainServer)
}