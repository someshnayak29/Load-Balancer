package cmd

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// iota is 0 rest all in declaration takes incremental value
const (
	Attempts int = iota
	Retry
)

// struct for backend created from server
type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
	Latency      int64
	Weight       float64
	Connections  int
}

func (b *Backend) SetAlive(alive bool) {

	b.mux.Lock() // write lock
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) IsActive() (alive bool) {

	b.mux.RLock() // read lock
	alive = b.Alive
	b.mux.RUnlock()
	return
}

func (b *Backend) SetLatency(latency int64) {
	if b.Alive {
		b.Latency = latency
	}
}

// check latency and status of connection by sending connection req over tcp

func getBackendStatus(u *url.URL) (bool, int64) {

	timeout := 2 * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	duration := time.Since(start).Microseconds()

	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false, int64(0)
	}

	defer conn.Close()
	return true, duration
}

// get attempts and retry value from context if set

func GetAttemptsFromContext(r *http.Request) int {

	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}

	return 1
}

func GetRetryFromContext(r *http.Request) int {

	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}

	return 0
}
