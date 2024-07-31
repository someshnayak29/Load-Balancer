package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/someshnayak29/load-balancer/cmd"
)

var (
	algorithm     = ""
	strict        = false
	ips           = []string{}
	xssProtection = true
)

type InternalConnections struct {
	mu          sync.Mutex
	connections map[string]int
}

var serverPool cmd.ServerPool
var internalConnections InternalConnections

func (i *InternalConnections) IncreaseConnection(key string) {

	i.mu.Lock()
	i.connections[key] += 1
	i.mu.Unlock()
}

func lb(w http.ResponseWriter, r *http.Request) {

	// if ip of server if among restricted ips then send internal server error
	if strict {
		ips, _ = cmd.ReadLines("config/iplists.txt")
		for _, ip := range ips {
			if ip == r.RemoteAddr {
				w.WriteHeader(500)
				return
			}
		}
	}

	var peer *cmd.Backend
	attempts := cmd.GetAttemptsFromContext(r)

	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	switch algorithm {
	case "least-time":
		peer = serverPool.GetLowestLatency()
	case "weighted-round-robin":
		peer = serverPool.GetHighestWeight()
	case "connection-per-time":
		peer = serverPool.GetNextPeer()
		internalConnections.IncreaseConnection(peer.URL.String())
	default:
		peer = serverPool.GetNextPeer()
	}

	if peer != nil {

		if internalConnections.connections[peer.URL.String()] > peer.Connections {
			peer.SetAlive(false)
			return
		}
		log.Printf("MESSAGE FORWARDED to server: %s\n", peer.URL)

		if xssProtection {
			w.Header().Set("X-XSS-Protection", " 1; mode=block")
		}

		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}

	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func healthCheck() {

	t := time.NewTicker(time.Minute * 1)
	for {
		select {
		case <-t.C:
			log.Println("Starting Health Check...")
			serverPool.HealthCheck()
			log.Println("Health Check Completed")
		}
	}
}

func resetConnections() {

	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-t.C:
			internalConnections.connections = serverPool.InitConnections()
		}
	}

}

func main() {

	var c cmd.Config
	c.GetConf() // get configuration details of servers and loadbalancer

	// for logging
	if c.Log {
		f, err := os.OpenFile("logs/tb.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666) // open for both read & write, if not available create it and data is appended to file
		// 0666: Sets the file permissions to be readable and writable by the owner, group, and others.
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f) // SetOutput sets the output destination for the standard logger, now all logs will be set here
	}

	algorithm = c.Algorithm
	strict = c.Strict
	xssProtection = c.XssProtection

	if len(c.Servers) == 0 {
		log.Fatal("Please provide one or more backends to load balancer inside the config.yaml file!!!")
	}

	// if strict true then get all restricted ip lists
	if strict {
		_, err := cmd.ReadLines("config/iplists.txt")
		if err != nil {
			panic(err)
		}
	}

	for _, server := range c.Servers {

		serverUrl, err := url.Parse(server.Host) // Parse parses a raw url into a [URL] structure.
		if err != nil {
			log.Fatal(err)
		}
		// Reverse proxy is used by load balancers to distribute requests different servers
		proxy := httputil.NewSingleHostReverseProxy(serverUrl)

		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {

			log.Printf("[%s] %s\n", serverUrl.Host, e.Error())
			retries := cmd.GetRetryFromContext(r)
			// again retry at same server after 1o ms
			if retries < 3 {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(r.Context(), cmd.Retry, retries+1)
					proxy.ServeHTTP(w, r.WithContext(ctx)) // again send req to same server
				}
			}

			serverPool.MarkBackendStatus(serverUrl, false)
			attempts := cmd.GetAttemptsFromContext(r)
			log.Printf("%s(%s) Attempting retry %d\n", r.RemoteAddr, r.URL.Path, attempts)

			// send req to diff servers
			ctx := context.WithValue(r.Context(), cmd.Attempts, attempts+1)
			lb(w, r.WithContext(ctx))
		}

		serverPool.AddBackend(
			&cmd.Backend{
				URL:          serverUrl,
				Alive:        true,
				ReverseProxy: proxy,
				Weight:       server.Weight,
				Connections:  server.Connections,
			},
		)
		log.Printf("Configured server: %s\n", serverUrl)
	}
	internalConnections.connections = serverPool.InitConnections()

	// starting the server i.e. load balancer server
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: http.HandlerFunc(lb),
	}

	// The health check is performed every minute concurrently
	go healthCheck()

	// Runs every minute to reset the internal connections map, if we want to reset connections every minute
	go resetConnections()

	log.Printf("Load Balancer started at :%d\n", c.Port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
