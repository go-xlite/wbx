package webcast

import (
	"sync"
	"time"
)

// SSEClientManager handles client connections for a specific SSE endpoint
type SSEClientManager struct {
	clients map[string]chan string
	mutex   sync.RWMutex
	stats   SSEStats
}

func newSSEClientManager() *SSEClientManager {
	return &SSEClientManager{
		clients: make(map[string]chan string),
		stats:   SSEStats{},
	}
}

func (scm *SSEClientManager) addClient(clientID string) chan string {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	client := make(chan string, 10)
	scm.clients[clientID] = client

	scm.stats.TotalConnections++
	scm.stats.CurrentConnections++
	scm.stats.LastConnectionTime = time.Now()

	return client
}

func (scm *SSEClientManager) removeClient(clientID string) {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	if client, exists := scm.clients[clientID]; exists {
		close(client)
		delete(scm.clients, clientID)

		scm.stats.CurrentConnections--
		scm.stats.LastDisconnectionTime = time.Now()
	}
}

func (scm *SSEClientManager) broadcast(message string) int {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	sentCount := 0
	for clientID, client := range scm.clients {
		select {
		case client <- message:
			sentCount++
		default:
			// Client buffer full, remove it asynchronously
			go scm.removeClient(clientID)
		}
	}

	scm.stats.MessagesSent += int64(sentCount)
	return sentCount
}

func (scm *SSEClientManager) sendToClient(clientID string, message string) bool {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	client, exists := scm.clients[clientID]
	if !exists {
		return false
	}

	select {
	case client <- message:
		scm.stats.MessagesSent++
		return true
	default:
		// Client buffer full
		go scm.removeClient(clientID)
		return false
	}
}

func (scm *SSEClientManager) getClientCount() int {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()
	return len(scm.clients)
}

func (scm *SSEClientManager) getStats() SSEStats {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()
	return scm.stats
}

func (scm *SSEClientManager) getClients() []string {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	clients := make([]string, 0, len(scm.clients))
	for id := range scm.clients {
		clients = append(clients, id)
	}
	return clients
}

func (scm *SSEClientManager) shutdown() {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	for clientID, client := range scm.clients {
		close(client)
		delete(scm.clients, clientID)
	}

	scm.stats.CurrentConnections = 0
	scm.stats.LastDisconnectionTime = time.Now()
}

func (scm *SSEClientManager) incrementRejections() {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()
	scm.stats.ConnectionsRejected++
}
