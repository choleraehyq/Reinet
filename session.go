package reinet

import (
	"container/list"
	"time"
	"net/url"
	"net/http"
	"encoding/base64"
	"crypto/rand"
	"io"
	"fmt"
	"sync"
)

type Manager struct {
	cookieName string
	lock sync.Mutex
	provider Provider
	maxLifeTime int64
}

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string) error
	SessionGC(maxLifeTime int64) 
}

type Session interface {
	Set(key, value interface{}) error
	Get(key interface{}) interface{}
	Delete(key interface{}) error
	SessionID() string
}

var provides map[string]Provider
var pder Provider

func init() {
	provides = make(map[string]Provider)
	pder = &DefaultProvider {
		list: list.New()
	}
	AddProvider("default", pder)
}

func NewManager(provideName, cookiename string, maxLifeTime int64) (*Manager, error) {
	provider, ok := provides[provideName]
	if !ok {
		return nil, fmt.Errorf("session: unknown provide %q (forgotten import?)", provideName)
	}
	return &Manager {
		provider: provider, 
		cookieName: cookieName,
		maxLifeTime: maxLifeTime,
	}, nil
}

func AddProvider(providerName string, provider Provider) {
	provides[provideName] = provider
}

func (self *Manager) sessionID() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (self *Manager) SessionStart(w http.ResponseWriter, r http.Request) (session Session) {
	self.lock.Lock()
	defer self.lock.Unlock()
	cookie, err := r.Cookie(self.cookieName)
	if err != nil || cookie.Value == "" {
		sid := self.sessionID()
		session, _ = self.provider.SessionInit(sid)
		cookie := http.Cookie{
			Name: self.cookieName,
			Value: url.QueryEscape(sid),
			Path: "/",
			HttpOnly: true,
			MaxAge: int(self.maxLifeTime),
		}
		http.SetCookie(w, &cookie)
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session, _ = self.provider.SessionRead(sid)
	}
	return
}

func (self *Manager) SessionDestroy(w http.ResponseWriter, r http.Request) {
	cookie, err := r.Cookie(self.cookieName)
	if err != nil || cookie.Value == "" {
		return 
	} else {
		self.lock.Lock()
		defer self.lock.Unlock()
		self.provider.SessionDestroy(cookie.Value)
		expiration := time.Now()
		cookie := http.Cookie {
			Name: self.cookieName,
			Path: "/",
			HttpOnly: true,
			Expires: expiration, 
			MaxAge: -1,
		}
		http.SetCookie(w, &cookie)
	}
}

func (self *Manager) GC() {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.provider.SessionGC(self.maxLifeTime)
	time.AfterFunc(time.Duration(self.maxLifeTime), func () {
		self.GC()
	})
}

type DefaultProvider struct {
	lock sync.Mutex
	sessions map[string]*list.Element
	list *list.List
}

type SessionStore struct {
	sid string
	timeAccessed time.Time
	value map[interface{}]interface{}
}

func (self *SessionStore) Set(key, value interface{}) error {
	self.value[key] = value
	pder.SessionUpdate(self.sid)
	return nil
}

func (self *SessionStore) Get(key interface{}) interface{} {
	pder.SessionUpdate(self.sid)
	if v, ok := self.value[key]; ok {
		return v
	} else {
		return nil
	}
	return nil
}

func (self *SessionStore) Delete(key interface{}) error {
	delete(self.value, key)
	pder.SessionUpdate(self.sid)
	return nil
}

func (self *SessionStore) SessionID() string {
	return self.sid
}

func (self *DefaultProvider) SessionInit(sid string) (Session, error){
	self.lock.Lock()
	defer self.lock.Unlock()
	v := make(map[interface{}]interface{}, 0)
	newsess := &SessionStore {
		sid: sid,
		timeAccessed: time.Now(),
		value: v,
	}
	element := self.list.PushBack(newsess)
	self.sessions[sid] = element
	return newsess, nil
}

func (self *DefaultProvider) SessionRead(sid string) (Session, error) {
	if element, ok := self.sessions[sid]; ok {
		return element.Value.(*SessionStore), nil
	} else {
		sess, err := self.SessionInit(sid)
		return sess, err
	}
	return nil, nil
}

func (self *DefaultProvider) SessionDestroy(sid string) error {
	if element, ok := self.sessions[sid]; ok {
		delete(self.sessions, sid)
		self.list.Remove(element)
		return nil
	}
	return nil
}

func (self *DefaultProvider) SessionGC(maxLifeTime int64) {
	self.lock.Lock()
	defer self.lock.Unlock()
	
	for {
		element := self.list.Back()
		if element == nil {
			break
		}
		if (element.Value.(*SessionStore).timeAccessed.Unix() + maxLifeTime) < time.Now().Unix() {
			self.list.Remove(element)
			delete(self.sessions, element.Value.(*SessionStore).sid)
		} else {
			break
		}
	}
} 

func (self *DefaultProvider) SessionUpdate(sid string) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	if element, ok := self.sessions[sid]; ok {
		element.Value.(*SessionStore).timeAccessed = time.Now()
		self.list.MoveToFront(element)
		return nil
	}
	return nil
}
