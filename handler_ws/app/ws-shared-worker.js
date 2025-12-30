
/**
 * Shared Worker WebSocket implementation
 * This worker maintains a persistent WebSocket connection shared across tabs/windows
 */

// Helper function for consistent timestamp logging
function timestamp() {
    const now = new Date();
    return `[${now.toISOString().substr(11, 12)}]`;
}

// Worker message types (must match browser-ws-manager.js)
const WORKER_CONNECT = 1;
const WORKER_DISCONNECT = 2;
const WORKER_SEND = 4;
const WORKER_CONNECTED = 8;
const WORKER_DISCONNECTED = 16;
const WORKER_MESSAGE = 32;
const WORKER_ERROR = 64;
const WORKER_RECONNECT = 128;
const WORKER_GLOBAL_SHUTDOWN = 256;

// Track all connected clients
const clients = new Set();

// Store WebSocket instance
let socket = null;
let connectionId = null;
let reconnectAttempts = 0;
let reconnectTimeout = null;
let givenUp = false;
let isReconnecting = false;
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
            type: WORKER_CONNECTED,
            connectionId: connectionId
        });
    }
};

// Handle messages from clients
function handleClientMessage(message, sourcePort) {
    console.log(`${timestamp()} [SharedWorker] Received message type ${message.type}`);
    switch (message.type) {
        case WORKER_CONNECT:
            console.log(`${timestamp()} [SharedWorker] WORKER_CONNECT - socket state: ${socket ? socket.readyState : 'null'}`);
            if (!socket || socket.readyState !== WebSocket.OPEN) {
                console.log(`${timestamp()} [SharedWorker] Resetting state for explicit connect`);
                givenUp = false; // Reset on explicit connect request
                reconnectAttempts = 0;
                clearTimeout(reconnectTimeout);
                isReconnecting = false;
                connectWebSocket(message.connectionId);
            }
            break;
            
        case WORKER_DISCONNECT:
            console.log(`${timestamp()} [SharedWorker] WORKER_DISCONNECT requested`);
            if (socket) {
                socket.close();
                socket = null;
            }
            break;
            
        case WORKER_SEND:
            if (socket && socket.readyState === WebSocket.OPEN) {
                console.log(`${timestamp()} [SharedWorker] Sending data`);
                socket.send(message.data);
            } else {
                console.log(`${timestamp()} [SharedWorker] Cannot send - socket not connected`);
                sourcePort.postMessage({
                    type: WORKER_ERROR,
                    error: 'Socket not connected'
                });
            }
            break;
            
        case WORKER_RECONNECT:
            // This message type is only for broadcasting TO clients, not receiving FROM them
            // Clients should not send this - it's a status notification only
            console.log(`${timestamp()} [SharedWorker] ⚠ WARNING: Received WORKER_RECONNECT from client (invalid - this is a status message only)`)
            break;
            
        case WORKER_GLOBAL_SHUTDOWN:
            console.log('[SharedWorker] Global shutdown requested');
            // Close all connections
            if (socket) {
                socket.close();
                socket = null;
            }
            
            // Notify all clients
            broadcastToClients({
                type: WORKER_DISCONNECTED,
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
    console.log(`${timestamp()} [SharedWorker] connectWebSocket called - isReconnecting: ${isReconnecting}, attempts: ${reconnectAttempts}`);
    
    if (socket && (socket.readyState === WebSocket.CONNECTING || socket.readyState === WebSocket.OPEN)) {
        console.log(`${timestamp()} [SharedWorker] Socket already connecting/connected, aborting`);
        return;
    }
    
    connectionId = connId || generateConnectionId();
    
    const protocol = self.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = self.location.host;
    const endpoint = self.location.pathname.split('/').slice(0, -2).join('/') + "/connect";
    

    const url = `${protocol}//${host}${endpoint}?connid=${connectionId}`;
    console.log(`${timestamp()} [SharedWorker] Creating WebSocket to ${url}`);
    
    try {
        socket = new WebSocket(url);
        
        socket.onopen = function() {
            console.log(`${timestamp()} [SharedWorker] ✓ WebSocket CONNECTED`);
            console.log(`${timestamp()} [SharedWorker] Resetting reconnectAttempts: ${reconnectAttempts} -> 0, isReconnecting: ${isReconnecting} -> false`);
            reconnectAttempts = 0;
            givenUp = false;
            isReconnecting = false;
            clearTimeout(reconnectTimeout);
            
            // Notify all clients that the connection is established
            broadcastToClients({
                type: WORKER_CONNECTED,
                connectionId: connectionId
            });
        };
        
        socket.onclose = function() {
            console.log(`${timestamp()} [SharedWorker] ✗ WebSocket CLOSED - isReconnecting was: ${isReconnecting}, attempts: ${reconnectAttempts}`);
            socket = null;
            
            // Notify all clients
            broadcastToClients({
                type: WORKER_DISCONNECTED
            });
            
            // Reset the reconnecting flag - this connection attempt is complete
            console.log(`${timestamp()} [SharedWorker] Setting isReconnecting: ${isReconnecting} -> false`);
            isReconnecting = false;
            
            // Schedule the next reconnection attempt
            console.log(`${timestamp()} [SharedWorker] Checking reconnect conditions - clients: ${clients.size}, givenUp: ${givenUp}`);
            if (clients.size > 0 && !givenUp) {
                reconnectWithBackoff();
            } else {
                console.log(`${timestamp()} [SharedWorker] Skipping reconnect`);
            }
        };
        
        socket.onerror = function(error) {
            console.error(`${timestamp()} [SharedWorker] ⚠ WebSocket ERROR:`, error);
            
            // Only broadcast errors if we haven't given up
            if (!givenUp) {
                broadcastToClients({
                    type: WORKER_ERROR,
                    error: 'WebSocket error'
                });
            }
            
            // Don't handle reconnection here - onclose will handle it
        };
        
        socket.onmessage = function(event) {
            // Forward messages to all connected clients
            // Split on newlines in case server batches messages
            const messages = event.data.split('\n').filter(msg => msg.trim());
            messages.forEach(message => {
                broadcastToClients({
                    type: WORKER_MESSAGE,
                    data: message
                });
            });
        };
    } catch (error) {
        console.error(`${timestamp()} [SharedWorker] ⚠ EXCEPTION creating WebSocket:`, error);
        socket = null;
        
        if (!givenUp) {
            broadcastToClients({
                type: WORKER_ERROR,
                error: 'Failed to create WebSocket connection'
            });
        }
        
        // Reset reconnecting flag since this connection attempt failed immediately
        console.log(`${timestamp()} [SharedWorker] Setting isReconnecting: ${isReconnecting} -> false (catch)`);
        isReconnecting = false;
        
        console.log(`${timestamp()} [SharedWorker] Checking reconnect after catch - clients: ${clients.size}, givenUp: ${givenUp}`);
        if (clients.size > 0 && !givenUp) {
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
    console.log(`${timestamp()} [SharedWorker] ══════ reconnectWithBackoff CALLED ══════`);
    console.log(`${timestamp()} [SharedWorker] State: isReconnecting=${isReconnecting}, attempts=${reconnectAttempts}, givenUp=${givenUp}`);
    
    // Prevent multiple simultaneous reconnection scheduling
    if (isReconnecting) {
        console.log(`${timestamp()} [SharedWorker] ⊘ BLOCKED: Reconnection already scheduled, skipping`);
        return;
    }
    
    if (reconnectAttempts >= maxReconnectAttempts) {
        console.log(`${timestamp()} [SharedWorker] ⊗ GIVING UP: Maximum reconnection attempts reached`);
        givenUp = true;
        broadcastToClients({
            type: WORKER_ERROR,
            error: 'Maximum reconnection attempts reached'
        });
        return;
    }
    
    // Mark that we're scheduling a reconnection
    console.log(`${timestamp()} [SharedWorker] Setting isReconnecting: false -> true`);
    isReconnecting = true;
    
    // Base delay of 2 seconds, exponential backoff with max of 60 seconds
    const baseDelay = 2000 * Math.pow(2, reconnectAttempts);
    const maxDelay = 60000;
    
    // Add jitter of ±20% to prevent thundering herd problem
    const jitter = 0.8 + (Math.random() * 0.4); // Random value between 0.8 and 1.2
    const delay = Math.min(Math.floor(baseDelay * jitter), maxDelay);
    
    reconnectAttempts++;
    
    console.log(`${timestamp()} [SharedWorker] ⏲ SCHEDULING reconnect: attempt ${reconnectAttempts} in ${delay}ms (base: ${baseDelay}ms, jitter: ${jitter.toFixed(2)})`);
    
    // No RECONNECTING constant defined, using WORKER_ERROR with reconnecting info
    broadcastToClients({
        type: WORKER_RECONNECT,
        attempt: reconnectAttempts,
        delay: delay
    });
    
    clearTimeout(reconnectTimeout);
    reconnectTimeout = setTimeout(() => {
        // Still keep flag true during connection attempt
        console.log(`${timestamp()} [SharedWorker] ▶ TIMEOUT FIRED: Executing reconnect attempt ${reconnectAttempts}`);
        connectWebSocket(connectionId);
    }, delay);
}

// Generate a unique connection ID
function generateConnectionId() {
    return Date.now().toString() + Math.random().toString(36).substring(2, 9);
}


