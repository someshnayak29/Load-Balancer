package cmd

import (
	"net/url"
	"sync/atomic"
)

type ServerPool struct {
	backends []*Backend
	current  uint64
}

// function to reset connection in slice of all servers
func (s *ServerPool) InitConnections() map[string]int {

	connections := make(map[string]int, len(s.backends))

	for _, server := range s.backends {
		server.Alive = true
		connections[server.URL.String()] = 0
	}

	return connections
}

// add backend to pool
func (s *ServerPool) AddBackend(backend *Backend) {
	s.backends = append(s.backends, backend)
}

// find next index to current server for round robin algo

func (s *ServerPool) NextIndex() int {
	// add 1 to current then % len of backends slice
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

func (s *ServerPool) MarkBackendStatus(backendUrl *url.URL, alive bool) {

	for _, server := range s.backends {
		if server.URL.String() == backendUrl.String() {
			server.SetAlive(alive)
			break
		}
	}
}

func (s *ServerPool) HealthCheck() {

	for _, server := range s.backends {

		alive, latency := getBackendStatus(server.URL)
		server.SetAlive(alive)
		server.SetLatency(latency)
	}
}

// Getnext active server -> this is for round robin algo

func (s *ServerPool) GetNextPeer() *Backend {

	next := s.NextIndex()
	l := len(s.backends) + next // so that after last index it comes back to first one

	for i := next; i < l; i++ {

		idx := i % len(s.backends)
		if s.backends[idx].IsActive() {

			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}

			return s.backends[idx]
		}
	}

	return nil
}

func (s *ServerPool) GetLowestLatency() *Backend {
	lowestServer := s.backends[0]

	for _, server := range s.backends[1:] {

		if server.Latency < lowestServer.Latency {
			lowestServer = server
		}
	}

	return lowestServer
}

func (s *ServerPool) GetHighestWeight() *Backend {
	highestWeight := s.backends[0]

	for _, server := range s.backends[1:] {

		if server.Weight > highestWeight.Weight {
			highestWeight = server
		}
	}

	return highestWeight
}
