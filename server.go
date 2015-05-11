package reinet

import (
	"regexp"
	"log"
	"strings"
	"http"
	"reflect"
	"strconv"
	"os"
)

type ReiServer struct {
	routes []route
	logger *log.Logger
	staticDir *map[string]string
	beforeFunc []handler
	afterFunc []handler
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
		
		ctx := &Context {
			req: r,
			res: w,
			formParams: formParams,
			urlQueryParams: urlQueryParams,
		}
		
		params := make([]reflect.Value, 0)
		
		if requireContext(reflect.TypeOf(route.handler)) {
			params = append(params, reflect.ValueOf(ctx))
		}
		
		for _, match := range matches[1:] {
			params = append(params, reflect.ValueOf(match))
		}
		
		self.execBeforeFunc(ctx)
		
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
		
		self.execAfterFunc(ctx)
		
		started = true
		break
	}
	
	if started == false {
		http.NotFound(w, r)
	}
}

func (self *ReiServer) addBefore(mid handler) {
	self.beforeFunc = append(self.beforeFunc, mid)
}

func (self *ReiServer) addAfter(mid handler) {
	self.afterFunc = append(self.afterFunc, mid)
}

func (self *ReiServer) execBeforeFunc(ctx *Context) {
	for _, f range self.beforeFunc {
		params := make([]reflect.Value, 0)
		if requireContext(reflect.TypeOf(f)) {
			params = append(params, reflect.ValueOf(ctx))
		}
		reflect.ValueOf(f).Call(params)
	}
}

func (self *ReiServer) execAfterFunc(ctx *Context) {
	for _, f range self.afterFunc {
		params := make([]reflect.Value, 0)
		if requireContext(reflect.TypeOf(f)) {
			params = append(params, reflect.ValueOf(ctx))
		}
		reflect.ValueOf(f).Call(params)
	}
}