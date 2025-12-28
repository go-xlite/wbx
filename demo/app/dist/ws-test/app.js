let sentCount = 0;
let receivedCount = 0;
let connectTime = null;
let uptimeInterval = null;

// Wait for wsManager to be available
function waitForWsManager(callback) {
    if (window.wsManager) {
        callback();
    } else {
        setTimeout(() => waitForWsManager(callback), 100);
    }
}

function escapeHtml(unsafe) {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

function updateStatus(status) {
    const statusEl = document.getElementById('wsStatus');
    statusEl.className = 'ws-status ' + status;
    if (status === 'connected') {
        statusEl.textContent = '🟢 Connected to WebSocket';
    } else if (status === 'disconnected') {
        statusEl.textContent = '🔴 Disconnected';
    } else if (status === 'connecting') {
        statusEl.textContent = '🟡 Connecting...';
    }
}

function addMessage(msg, type = 'info') {
    const messagesDiv = document.getElementById('messages');
    const msgDiv = document.createElement('div');
    msgDiv.className = 'message ' + type;
    const timestamp = new Date().toLocaleTimeString();
    msgDiv.innerHTML = `<strong>[${timestamp}]</strong> ${escapeHtml(msg)}`;
    messagesDiv.appendChild(msgDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

function updateStats() {
    document.getElementById('sentCount').textContent = sentCount;
    document.getElementById('receivedCount').textContent = receivedCount;
}

function updateUptime() {
    if (connectTime) {
        const elapsed = Math.floor((Date.now() - connectTime) / 1000);
        const hours = Math.floor(elapsed / 3600);
        const minutes = Math.floor((elapsed % 3600) / 60);
        const seconds = elapsed % 60;
        document.getElementById('uptime').textContent = 
            `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    } else {
        document.getElementById('uptime').textContent = '00:00:00';
    }
}

function connect() {
    if (!window.wsManager) {
        addMessage('WebSocket manager not loaded yet', 'error');
        return;
    }

    if (window.wsManager.connectionState === 'connected') {
        addMessage('Already connected', 'error');
        return;
    }

    updateStatus('connecting');
    addMessage('Connecting via WebSocket manager...', 'info');
    addMessage(`Connection mode: ${window.wsManager.connectionMode || 'auto-detect'}`, 'info');

    // Connect using the WebSocket manager
    window.wsManager.connect();
}

function disconnect() {
    if (!window.wsManager) {
        addMessage('WebSocket manager not loaded', 'error');
        return;
    }

    if (window.wsManager.connectionState !== 'connected') {
        addMessage('Not connected', 'error');
        return;
    }

    addMessage('Disconnecting...', 'info');
    window.wsManager.disconnect();
}

function sendMessage() {
    if (!window.wsManager) {
        addMessage('WebSocket manager not loaded!', 'error');
        return;
    }

    // Can send if connected OR if using coordination (secondary tab can relay through primary)
    const canSend = window.wsManager.connectionState === 'connected' || 
                    (window.wsManager.broadcastChannel && window.wsManager.broadcastChannel !== null);
    
    if (!canSend) {
        addMessage('Not connected!', 'error');
        return;
    }

    const input = document.getElementById('messageInput');
    const message = input.value.trim();
    
    if (!message) {
        addMessage('Cannot send empty message', 'error');
        return;
    }

    window.wsManager.send(message);
    sentCount++;
    updateStats();
    addMessage('Sent: ' + message, 'sent');
    input.value = '';
}

function sendPing() {
    if (!window.wsManager) {
        addMessage('WebSocket manager not loaded!', 'error');
        return;
    }

    // Can send if connected OR if using coordination
    const canSend = window.wsManager.connectionState === 'connected' || 
                    (window.wsManager.broadcastChannel && window.wsManager.broadcastChannel !== null);
    
    if (!canSend) {
        addMessage('Not connected!', 'error');
        return;
    }

    const ping = JSON.stringify({ type: 'ping', timestamp: Date.now() });
    window.wsManager.send(ping);
    sentCount++;
    updateStats();
    addMessage('Sent PING', 'sent');
}

function sendBurst() {
    if (!window.wsManager) {
        addMessage('WebSocket manager not loaded!', 'error');
        return;
    }

    // Can send if connected OR if using coordination
    const canSend = window.wsManager.connectionState === 'connected' || 
                    (window.wsManager.broadcastChannel && window.wsManager.broadcastChannel !== null);
    
    if (!canSend) {
        addMessage('Not connected!', 'error');
        return;
    }

    for (let i = 1; i <= 10; i++) {
        const msg = `Burst message ${i}/10`;
        window.wsManager.send(msg);
        sentCount++;
    }
    updateStats();
    addMessage('Sent burst of 10 messages', 'sent');
}

function clearMessages() {
    document.getElementById('messages').innerHTML = '';
    addMessage('Message log cleared', 'info');
}

// Enter key to send
document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('messageInput').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            sendMessage();
        }
    });

    // Initial state
    updateStatus('disconnected');
    updateStats();
    updateUptime();

    // Setup WebSocket manager event handlers
    waitForWsManager(function() {
        addMessage('WebSocket manager loaded', 'info');
        addMessage(`Connection ID: ${window.wsManager.connectionId}`, 'info');

        // Handle connection open
        window.wsManager.on('open', function() {
            updateStatus('connected');
            addMessage('Connected successfully!', 'received');
            addMessage(`Connection mode: ${window.wsManager.connectionMode}`, 'info');
            connectTime = Date.now();
            uptimeInterval = setInterval(updateUptime, 1000);
            document.getElementById('connectBtn').disabled = true;
            document.getElementById('disconnectBtn').disabled = false;
            document.getElementById('sendBtn').disabled = false;
            document.getElementById('pingBtn').disabled = false;
            document.getElementById('burstBtn').disabled = false;
        });

        // Handle incoming messages
        window.wsManager.on('message', function(data) {
            receivedCount++;
            updateStats();
            addMessage('Received: ' + data, 'received');
        });

        // Handle connection close - only disconnect UI if truly disconnected
        window.wsManager.on('close', function() {
            // Only show disconnected if we're not using SharedWorker coordination
            // or if the SharedWorker itself is disconnected
            if (!window.wsManager.broadcastChannel || window.wsManager.connectionMode === 'direct') {
                updateStatus('disconnected');
                addMessage('Connection closed', 'error');
                connectTime = null;
                if (uptimeInterval) {
                    clearInterval(uptimeInterval);
                    uptimeInterval = null;
                }
                updateUptime();
                document.getElementById('connectBtn').disabled = false;
                document.getElementById('disconnectBtn').disabled = true;
                document.getElementById('sendBtn').disabled = true;
                document.getElementById('pingBtn').disabled = true;
                document.getElementById('burstBtn').disabled = true;
            }
        });

        // Handle errors
        window.wsManager.on('error', function(error) {
            addMessage('WebSocket error occurred', 'error');
            console.error('WebSocket error:', error);
        });

        // Handle coordination events (when using SharedWorker)
        if (window.wsManager.onCoordinationEvent) {
            window.wsManager.onCoordinationEvent('became_primary', function() {
                addMessage('This tab became the primary connection', 'info');
            });

            window.wsManager.onCoordinationEvent('became_secondary', function() {
                addMessage('This tab is now secondary (connection maintained via SharedWorker)', 'info');
                // Keep the connected state - we're still connected via SharedWorker
                updateStatus('connected');
            });

            window.wsManager.onCoordinationEvent('primary_disconnected', function() {
                // Primary connection lost, but we might take over
                addMessage('Primary connection lost', 'error');
            });

            window.wsManager.onCoordinationEvent('all_disconnected', function() {
                // All connections truly lost
                updateStatus('disconnected');
                addMessage('All connections closed', 'error');
                connectTime = null;
                if (uptimeInterval) {
                    clearInterval(uptimeInterval);
                    uptimeInterval = null;
                }
                updateUptime();
                document.getElementById('connectBtn').disabled = false;
                document.getElementById('disconnectBtn').disabled = true;
                document.getElementById('sendBtn').disabled = true;
                document.getElementById('pingBtn').disabled = true;
                document.getElementById('burstBtn').disabled = true;
            });
        }
    });
});
