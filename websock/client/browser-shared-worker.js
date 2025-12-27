//{{ define "browser-worker" }}

/**
 * Shared Worker WebSocket implementation
 * This worker maintains a persistent WebSocket connection shared across tabs/windows
 */

// Track all connected clients
const clients = new Set();

// Store WebSocket instance
let socket = null;
let connectionId = null;
let reconnectAttempts = 0;
let reconnectTimeout = null;
const maxReconnectAttempts = 10;

// Handle messages from connected clients
self.onconnect = function(e) {
    const port = e.ports[0];
    clients.add(port);
    
    port.start();
    
    // Listen for messages from this client
    port.addEventListener('message', function(event) {
        handleClientMessage(event.data, port);
    });
    
    // Handle client disconnection
    port.addEventListener('close', function() {
        clients.delete(port);
        
        // If no more clients are connected, close the WebSocket
        if (clients.size === 0 && socket) {
            socket.close();
            socket = null;
            clearTimeout(reconnectTimeout);
        }
    });
    
    // If socket is already connected, inform the new client
    if (socket && socket.readyState === WebSocket.OPEN) {
        port.postMessage({
            type: 'CONNECTED',
            connectionId: connectionId
        });
    }
};

// Handle messages from clients
function handleClientMessage(message, sourcePort) {
    switch (message.type) {
        case 'CONNECT':
            if (!socket || socket.readyState !== WebSocket.OPEN) {
                connectWebSocket(message.connectionId);
            }
            break;
            
        case 'DISCONNECT':
            if (socket) {
                socket.close();
                socket = null;
            }
            break;
            
        case 'SEND':
            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.send(message.data);
            } else {
                sourcePort.postMessage({
                    type: 'ERROR',
                    error: 'Socket not connected'
                });
            }
            break;
            
        case 'RECONNECT':
            if (!socket || socket.readyState !== WebSocket.OPEN) {
                connectWebSocket(message.connectionId);
            }
            break;
            
        case 'GLOBAL_SHUTDOWN':
            console.log('[SharedWorker] Global shutdown requested');
            // Close all connections
            if (socket) {
                socket.close();
                socket = null;
            }
            
            // Notify all clients
            broadcastToClients({
                type: 'FORCE_DISCONNECT',
                reason: 'global_shutdown'
            });
            
            // Try to close all ports
            clients.forEach(port => {
                try {
                    port.close();
                } catch (e) {
                    console.error('[SharedWorker] Error closing port', e);
                }
            });
            
            // Clear clients
            clients.clear();
            
            // Attempt self-termination (may not work in all browsers)
            try {
                self.close();
            } catch (e) {
                console.error('[SharedWorker] Error self-terminating', e);
            }
            break;
    }
}

// Connect to the WebSocket server
function connectWebSocket(connId) {
    if (socket && (socket.readyState === WebSocket.CONNECTING || socket.readyState === WebSocket.OPEN)) {
        return;
    }
    
    connectionId = connId || generateConnectionId();
    
    const protocol = self.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = self.location.host;
    const url = `${protocol}//${host}{{ .Route }}?connid=${connectionId}`;
    
    try {
        socket = new WebSocket(url);
        
        socket.onopen = function() {
            console.log('[SharedWorker] WebSocket connected');
            reconnectAttempts = 0;
            
            // Notify all clients that the connection is established
            broadcastToClients({
                type: 'CONNECTED',
                connectionId: connectionId
            });
        };
        
        socket.onclose = function() {
            console.log('[SharedWorker] WebSocket closed');
            
            // Notify all clients
            broadcastToClients({
                type: 'DISCONNECTED'
            });
            
            // Attempt to reconnect if there are clients
            if (clients.size > 0) {
                reconnectWithBackoff();
            }
        };
        
        socket.onerror = function(error) {
            console.error('[SharedWorker] WebSocket error:', error);
            
            broadcastToClients({
                type: 'ERROR',
                error: 'WebSocket error'
            });
        };
        
        socket.onmessage = function(event) {
            // Forward messages to all connected clients
            broadcastToClients({
                type: 'MESSAGE',
                data: event.data
            });
        };
    } catch (error) {
        console.error('[SharedWorker] Error creating WebSocket:', error);
        
        broadcastToClients({
            type: 'ERROR',
            error: 'Failed to create WebSocket connection'
        });
        
        if (clients.size > 0) {
            reconnectWithBackoff();
        }
    }
}

// Send message to all connected clients
function broadcastToClients(message) {
    clients.forEach(client => {
        try {
            client.postMessage(message);
        } catch (error) {
            console.error('[SharedWorker] Error sending message to client:', error);
        }
    });
}

// Reconnect with exponential backoff
function reconnectWithBackoff() {
    if (reconnectAttempts >= maxReconnectAttempts) {
        console.log('[SharedWorker] Maximum reconnection attempts reached');
        broadcastToClients({
            type: 'ERROR',
            error: 'Maximum reconnection attempts reached'
        });
        return;
    }
    
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
    reconnectAttempts++;
    
    console.log(`[SharedWorker] Attempting to reconnect in ${delay}ms (attempt ${reconnectAttempts})`);
    
    broadcastToClients({
        type: 'RECONNECTING',
        attempt: reconnectAttempts,
        delay: delay
    });
    
    clearTimeout(reconnectTimeout);
    reconnectTimeout = setTimeout(() => {
        connectWebSocket(connectionId);
    }, delay);
}

// Generate a unique connection ID
function generateConnectionId() {
    return Date.now().toString() + Math.random().toString(36).substring(2, 9);
}

// {{ end }}
