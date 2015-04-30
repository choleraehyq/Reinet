# Reinet
一个使用Golang编写的极简的Web框架,我编写此框架出于学习目的,未来我将不断维护它并加入新的功能.

##概况
Reinet借鉴了[web.go](https://github.com/hoisie/web)和[beego](https://github.com/astaxie/beego),没有为用户提供脚手架,同时API的设计也与web.go类似.目前Reinet支持以下特性:
* 使用正则表达式以及带有参数的URL的路由分发
* 静态文件的服务
* 基本的会话管理

##安装
Reinet是在Ubuntu15.04上基于Go1.4.2开发并测试的,目前尚未在其他平台上测试过.
安装Reinet:

`go get github.com/choleraehyq/reinet`

从源码编译:
```
git clone git://github.com/choleraehyq/reinet
cd reinet && go build 
```
##样例程序
```Golang
package main
import (
    "github.com/choleraehyq/reinet"
)
func hello(id string) string {
    return "Hello, " + id
}
func main() {
    reinet.Get("/:id([1-9]+)", hello)
    reinet.Run(":1234")
}
```
执行`go run hello.go`运行程序.
之后访问http://localhost:1234/(一个数字作为id)即可.

##API
```
func Get(pattern string, handleFunc handler)
func Post(pattern string, handleFunc handler)
func Delete(pattern string, handleFunc handler)
func Patch(pattern string, handleFunc handler)
func Put(pattern string, handleFunc handler)
func GivenMethod(pattern string, handleFunc handler, method string)
func SetStatic(url string, path string)
```
这是简单的路由注册函数.`GivenMethod`可以传入一个用户指定的方法并注册路由.`SetStatic`用于注册静态文件路由.

传入的handleFunc的返回值应当是string类型,返回值将作为页面内容发给客户端,如果函数发生了重定向那么应当返回nil.handleFunc的参数由两部分组成,首先是可选的`reinet.Context`类型参数,这个类型的定义如下:
```Golang
type Context struct {
	req *http.Request
	res http.ResponseWriter
	formParams map[string]string
	urlQueryParams map[string]string
}
```
包含了表单参数,url查询参数和这次请求本身.handleFunc如果有`reinet.Context`类型参数那么这个参数必须是第一个参数.第二部分是注册的url中带有的参数,这部分参数的数量必须与url中参数的数量一致.目前这些参数必须都是string类型,在url中以正则表达式的方式表示.

reinet提供了`Sessions`全局变量用于会话管理.支持的方法如下:
```
func (self *Manager) SessionStart(w http.ResponseWriter, r http.Request) (session Session) 
func (self *Manager) SessionDestroy(w http.ResponseWriter, r http.Request) 
```
`SessionStart`会返回一个会话,`SessionDestroy`则会销毁这个会话.

会话的类型是`Session`接口,定义如下:
```
type Session interface {
	Set(key, value interface{}) error
	Get(key interface{}) interface{}
	Delete(key interface{}) error
	SessionID() string
}
```
支持简单的增删改操作,同时可以返回这个会话的ID.

会话过期会被删除,目前默认的过期时间为3600秒.

目前版本的API尚不稳定,未来会开放更多API,并进一步支持用户定制和拓展.

##TODO
* 安全性有待加强,比如预防Session劫持,安全的cookie等
* 目前版本的代码已经考虑到了拓展性和定制性的要求,下一步需要设计并开放相关API
* ORM(MySQL,MongoDB)
* 模板的支持

##LICENSE
本项目基于MIT许可证开源