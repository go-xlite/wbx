import { createWebSocketManager, WS_EVENT, WS_STATE, WS_MODE, WS_COORD_EVENT, WS_SESSION_STRATEGY } from './ws-manager.js';

let wsManager = null;
let sentCount = 0;
let receivedCount = 0;
let connectTime = null;
let uptimeInterval = null;
let switchingMode = false;
// Load session strategy from sessionStorage (tab-specific) or use default
const storedStrategy = sessionStorage.getItem('ws-session-strategy');
let currentSessionStrategy = storedStrategy ? parseInt(storedStrategy, 10) : WS_SESSION_STRATEGY.SHARED;

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
    statusEl.className = 'ws-status ' + (status === WS_STATE.CONNECTED ? 'connected' : status === WS_STATE.CONNECTING ? 'connecting' : 'disconnected');
    if (status === WS_STATE.CONNECTED) {
        statusEl.textContent = '🟢 Connected to WebSocket';
    } else if (status === WS_STATE.DISCONNECTED) {
        statusEl.textContent = '🔴 Disconnected';
    } else if (status === WS_STATE.CONNECTING) {
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
    if (!wsManager) {
        addMessage('WebSocket manager not loaded yet', 'error');
        return;
    }

    if (wsManager.connectionState === WS_STATE.CONNECTED) {
        addMessage('Already connected', 'error');
        return;
    }

    updateStatus(WS_STATE.CONNECTING);
    addMessage('Connecting via WebSocket manager...', 'info');
    addMessage(`Connection mode: ${wsManager.connectionMode || 'auto-detect'}`, 'info');

    // Connect using the WebSocket manager
    wsManager.connect();
}

function disconnect() {
    if (!wsManager) {
        addMessage('WebSocket manager not loaded', 'error');
        return;
    }

    if (window.wsManager.connectionState !== WS_STATE.CONNECTED) {
        addMessage('Not connected', 'error');
        return;
    }

    addMessage('Disconnecting...', 'info');
    window.wsManager.disconnect();
}

function sendMessage() {
    if (!wsManager) {
        addMessage('WebSocket manager not loaded!', 'error');
        return;
    }

    if (!wsManager.canSend()) {
        addMessage('Not connected!', 'error');
        return;
    }

    const input = document.getElementById('messageInput');
    const message = input.value.trim();
    
    if (!message) {
        addMessage('Cannot send empty message', 'error');
        return;
    }

    wsManager.send(message);
    sentCount++;
    updateStats();
    addMessage('Sent: ' + message, 'sent');
    input.value = '';
}

function sendPing() {
    if (!wsManager) {
        addMessage('WebSocket manager not loaded!', 'error');
        return;
    }

    if (!wsManager.canSend()) {
        addMessage('Not connected!', 'error');
        return;
    }

    const ping = JSON.stringify({ type: 'ping', timestamp: Date.now() });
    wsManager.send(ping);
    sentCount++;
    updateStats();
    addMessage('Sent PING', 'sent');
}

function sendBurst() {
    if (!wsManager) {
        addMessage('WebSocket manager not loaded!', 'error');
        return;
    }

    if (!wsManager.canSend()) {
        addMessage('Not connected!', 'error');
        return;
    }

    for (let i = 1; i <= 3; i++) {
        const msg = `Burst message ${i}/3`;
        wsManager.send(msg);
        sentCount++;
    }
    updateStats();
    addMessage('Sent burst of 3 messages', 'sent');
}

function clearMessages() {
    document.getElementById('messages').innerHTML = '';
    addMessage('Message log cleared', 'info');
}

function setMode(mode) {
    if (!wsManager) {
        addMessage('WebSocket manager not loaded', 'error');
        return;
    }

    if (wsManager.switchMode(mode)) {
        addMessage(`Switched to ${wsManager.getModeName(mode)} mode`, 'info');
        // Update active button styling
        document.querySelectorAll('.mode-btn').forEach(btn => btn.classList.remove('active'));
        document.querySelector(`[data-mode="${mode}"]`).classList.add('active');
        
        // Handle UI state during mode switch
        if (wsManager.connectionState === WS_STATE.CONNECTED || wsManager.canSend()) {
            switchingMode = true;
            addMessage('Reconnecting with new mode...', 'info');
            setTimeout(() => {
                switchingMode = false;
            }, 1500);
        }
    } else {
        addMessage('Failed to set connection mode', 'error');
    }
}

function clearMode() {
    if (!wsManager) {
        addMessage('WebSocket manager not loaded', 'error');
        return;
    }

    wsManager.resetMode();
    addMessage('Cleared preferred mode (will auto-detect)', 'info');
    document.querySelectorAll('.mode-btn').forEach(btn => btn.classList.remove('active'));
    
    // Handle UI state during mode reset
    if (wsManager.connectionState === WS_STATE.CONNECTED) {
        switchingMode = true;
        addMessage('Reconnecting with auto-detect...', 'info');
        setTimeout(() => {
            switchingMode = false;
        }, 1500);
    }
}

function setSessionStrategy(strategy) {
    currentSessionStrategy = strategy;
    // Save to sessionStorage (tab-specific)
    sessionStorage.setItem('ws-session-strategy', strategy);
    
    const strategyNames = {
        [WS_SESSION_STRATEGY.ISOLATED]: 'Isolated',
        [WS_SESSION_STRATEGY.SHARED]: 'Shared'
    };
    
    addMessage(`Session strategy will be: ${strategyNames[strategy]} (Direct mode only, takes effect on next reload)`, 'info');
    document.getElementById('currentStrategy').textContent = strategyNames[strategy];
    
    // Update active button styling
    document.querySelectorAll('.session-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelector(`[data-strategy="${strategy}"]`).classList.add('active');
}

function updateSessionInfo() {
    if (!wsManager) return;
    
    const strategyNames = {
        [WS_SESSION_STRATEGY.ISOLATED]: 'Isolated',
        [WS_SESSION_STRATEGY.SHARED]: 'Shared',
        [WS_SESSION_STRATEGY.SHARED_CONNECTION]: 'Shared Connection (SharedWorker)'
    };
    
    const strategy = wsManager.sessionStrategy || currentSessionStrategy;
    document.getElementById('currentStrategy').textContent = strategyNames[strategy] || `Unknown (${strategy})`;
    document.getElementById('currentSessionId').textContent = wsManager.sessionId || '-';
    
    // Show tab count if coordination is enabled
    if (wsManager.isCoordinationEnabled && wsManager.isCoordinationEnabled()) {
        const tabs = wsManager.getKnownTabs ? wsManager.getKnownTabs() : [];
        document.getElementById('sessionTabCount').textContent = tabs.length;
    } else {
        document.getElementById('sessionTabCount').textContent = '1 (no coordination)';
    }
    
    // Update session data display
    updateSessionDataDisplay();
}

function updateSessionDataDisplay() {
    if (!wsManager) return;
    const display = document.getElementById('sessionDataDisplay');
    display.textContent = JSON.stringify(wsManager.sessionData || {}, null, 2);
}

function setSessionData() {
    if (!wsManager) {
        addMessage('WebSocket manager not loaded', 'error');
        return;
    }
    
    const key = document.getElementById('sessionKey').value.trim();
    const value = document.getElementById('sessionValue').value.trim();
    
    if (!key) {
        addMessage('Please enter a key', 'error');
        return;
    }
    
    wsManager.setSessionValue(key, value);
    addMessage(`Set session[${key}] = "${value}"`, 'info');
    document.getElementById('sessionKey').value = '';
    document.getElementById('sessionValue').value = '';
    updateSessionDataDisplay();
}

function getSessionData() {
    if (!wsManager) {
        addMessage('WebSocket manager not loaded', 'error');
        return;
    }
    
    const key = document.getElementById('sessionKey').value.trim();
    
    if (!key) {
        addMessage('Please enter a key to get', 'error');
        return;
    }
    
    const value = wsManager.getSessionValue(key);
    if (value !== undefined) {
        addMessage(`session[${key}] = "${value}"`, 'info');
        document.getElementById('sessionValue').value = value;
    } else {
        addMessage(`session[${key}] is not set`, 'error');
    }
}

function clearSessionData() {
    if (!wsManager) {
        addMessage('WebSocket manager not loaded', 'error');
        return;
    }
    
    if (confirm('Clear all session data?')) {
        wsManager.clearSession();
        addMessage('Session data cleared', 'info');
        updateSessionDataDisplay();
        updateSessionInfo();
    }
}

// Initialize the app
async function initApp() {
    // Attach event listeners to buttons
    document.getElementById('connectBtn').addEventListener('click', connect);
    document.getElementById('disconnectBtn').addEventListener('click', disconnect);
    document.getElementById('sendBtn').addEventListener('click', sendMessage);
    document.getElementById('pingBtn').addEventListener('click', sendPing);
    document.getElementById('burstBtn').addEventListener('click', sendBurst);
    document.querySelector('[data-mode="1"]').addEventListener('click', () => setMode(1));
    document.querySelector('[data-mode="4"]').addEventListener('click', () => setMode(4));
    
    // Session strategy buttons (only Isolated and Shared for Direct mode)
    document.querySelector('[data-strategy="1"]').addEventListener('click', () => setSessionStrategy(WS_SESSION_STRATEGY.ISOLATED));
    document.querySelector('[data-strategy="2"]').addEventListener('click', () => setSessionStrategy(WS_SESSION_STRATEGY.SHARED));
    
    // Session data buttons
    document.getElementById('setSessionBtn').addEventListener('click', setSessionData);
    document.getElementById('getSessionBtn').addEventListener('click', getSessionData);
    document.getElementById('clearSessionBtn').addEventListener('click', clearSessionData);
    
    // Find buttons by text content
    const buttons = document.querySelectorAll('button');
    buttons.forEach(btn => {
        if (btn.textContent.includes('Clear Log')) {
            btn.addEventListener('click', clearMessages);
        } else if (btn.textContent.includes('Auto-Detect')) {
            btn.addEventListener('click', clearMode);
        }
    });
    
    document.getElementById('messageInput').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            sendMessage();
        }
    });

    // Initial state
    updateStatus(WS_STATE.DISCONNECTED);
    updateStats();
    updateUptime();

    // Create WebSocket manager instance
    addMessage('Loading WebSocket manager...', 'info');
    wsManager = await createWebSocketManager({ 
        debug: true,
        endpoint: '/ws/connect',  // Namespace session by endpoint
        sessionStrategy: currentSessionStrategy
    });
    
    addMessage('WebSocket manager loaded', 'info');
    addMessage(`Connection ID: ${wsManager.connectionId}`, 'info');
    addMessage(`Session ID: ${wsManager.sessionId}`, 'info');
    
    // Update session info UI
    updateSessionInfo();
    
    // Highlight active session strategy button
    document.querySelector(`[data-strategy="${currentSessionStrategy}"]`).classList.add('active');

    // Highlight active mode button if a mode is set
    if (wsManager.connectionMode) {
        const modeBtn = document.querySelector(`[data-mode="${wsManager.connectionMode}"]`);
        if (modeBtn) {
            modeBtn.classList.add('active');
        }
        addMessage(`Using preferred mode: ${wsManager.getModeName()}`, 'info');
    }

    // Check if already connected when we load
    if (wsManager.connectionState === WS_STATE.CONNECTED) {
        updateStatus(WS_STATE.CONNECTED);
        addMessage('Already connected on load', 'info');
        addMessage(`Connection mode: ${wsManager.getModeName()}`, 'info');
        connectTime = Date.now();
        uptimeInterval = setInterval(updateUptime, 1000);
        document.getElementById('connectBtn').disabled = true;
        document.getElementById('disconnectBtn').disabled = false;
        document.getElementById('sendBtn').disabled = false;
        document.getElementById('pingBtn').disabled = false;
        document.getElementById('burstBtn').disabled = false;
    }

    // Handle connection open
    wsManager.on(WS_EVENT.OPEN, function() {
            updateStatus(WS_STATE.CONNECTED);
            addMessage('Connected successfully!', 'received');
            addMessage(`Connection mode: ${wsManager.getModeName()}`, 'info');
            connectTime = Date.now();
            uptimeInterval = setInterval(updateUptime, 1000);
            document.getElementById('connectBtn').disabled = true;
            document.getElementById('disconnectBtn').disabled = false;
            document.getElementById('sendBtn').disabled = false;
            document.getElementById('pingBtn').disabled = false;
            document.getElementById('burstBtn').disabled = false;
        });

        // Handle incoming messages
        wsManager.on(WS_EVENT.MESSAGE, function(data) {
            console.log('Raw message received:', JSON.stringify(data));
            receivedCount++;
            updateStats();
            addMessage('Received: ' + data, 'received');
        });

        // Handle connection close - only disconnect UI if truly disconnected
        wsManager.on(WS_EVENT.CLOSE, function() {
            // Don't update UI if we're switching modes
            if (switchingMode) {
                return;
            }
            
            // Only show disconnected if we're not using SharedWorker coordination
            // or if the SharedWorker itself is disconnected
            if (!wsManager.isCoordinationEnabled() || wsManager.connectionMode === WS_MODE.DIRECT) {
                updateStatus(WS_STATE.DISCONNECTED);
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
        wsManager.on(WS_EVENT.ERROR, function(error) {
            addMessage('WebSocket error occurred', 'error');
            console.error('WebSocket error:', error);
        });

        // Handle coordination events (when using SharedWorker)
        if (wsManager.onCoordinationEvent) {
            wsManager.onCoordinationEvent(WS_COORD_EVENT.BECAME_PRIMARY, function() {
                addMessage('This tab became the primary connection', 'info');
                // Update UI - check connection state after a short delay to ensure it's set
                setTimeout(() => {
                    if (wsManager.connectionState === WS_STATE.CONNECTED) {
                        updateStatus(WS_STATE.CONNECTED);
                        connectTime = connectTime || Date.now();
                        if (!uptimeInterval) {
                            uptimeInterval = setInterval(updateUptime, 1000);
                        }
                        document.getElementById('connectBtn').disabled = true;
                        document.getElementById('disconnectBtn').disabled = false;
                        document.getElementById('sendBtn').disabled = false;
                        document.getElementById('pingBtn').disabled = false;
                        document.getElementById('burstBtn').disabled = false;
                    }
                }, 100);
            });

            wsManager.onCoordinationEvent(WS_COORD_EVENT.BECAME_SECONDARY, function() {
                addMessage('This tab is now secondary (connection maintained via SharedWorker)', 'info');
                // Update UI - check connection state after a short delay
                setTimeout(() => {
                    if (wsManager.connectionState === WS_STATE.CONNECTED) {
                        updateStatus(WS_STATE.CONNECTED);
                        connectTime = connectTime || Date.now();
                        if (!uptimeInterval) {
                            uptimeInterval = setInterval(updateUptime, 1000);
                        }
                        document.getElementById('connectBtn').disabled = true;
                        document.getElementById('disconnectBtn').disabled = false;
                        document.getElementById('sendBtn').disabled = false;
                        document.getElementById('pingBtn').disabled = false;
                        document.getElementById('burstBtn').disabled = false;
                    }
                }, 100);
            });

            wsManager.onCoordinationEvent(WS_COORD_EVENT.TABS_UPDATED, function(tabs) {
                // Tab list updated
                console.log('Known tabs:', tabs);
                updateSessionInfo();
            });

            wsManager.onCoordinationEvent(WS_COORD_EVENT.ENABLED, function() {
                addMessage('Tab coordination enabled', 'info');
                // Check if already connected and update UI after a short delay
                setTimeout(() => {
                    if (wsManager.connectionState === WS_STATE.CONNECTED) {
                        updateStatus(WS_STATE.CONNECTED);
                        connectTime = connectTime || Date.now();
                        if (!uptimeInterval) {
                            uptimeInterval = setInterval(updateUptime, 1000);
                        }
                        document.getElementById('connectBtn').disabled = true;
                        document.getElementById('disconnectBtn').disabled = false;
                        document.getElementById('sendBtn').disabled = false;
                        document.getElementById('pingBtn').disabled = false;
                        document.getElementById('burstBtn').disabled = false;
                    }
                }, 100);
            });
        }
}

// Start the app - createWebSocketManager handles DOM ready internally
initApp();

