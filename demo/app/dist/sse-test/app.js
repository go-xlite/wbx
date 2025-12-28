/**
 * SSE Test Application
 * Uses sse-manager-stub.js for dynamic import with full intellisense
 */

import { createSSEManager } from './sse-manager-stub.js';

let sseManager = null;
let receivedCount = 0;
let reconnectAttempts = 0;
let connectTime = null;
let uptimeInterval = null;

function escapeHtml(unsafe) {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

function updateStatus(status) {
    const statusEl = document.getElementById('sseStatus');
    statusEl.className = 'sse-status ' + status;
    if (status === 'connected') {
        statusEl.textContent = '🟢 Connected to SSE';
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
    document.getElementById('receivedCount').textContent = receivedCount;
    document.getElementById('reconnects').textContent = reconnectAttempts;
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

async function connect() {
    if (sseManager) {
        addMessage('Already connected or connecting', 'error');
        return;
    }

    updateStatus('connecting');
    
    const pathParts = window.location.pathname.split('/').filter(p => p);
    const pathPrefix = pathParts.length > 0 ? '/' + pathParts[0] : '';
    
    const protocol = window.location.protocol;
    const sseUrl = `${protocol}//${window.location.host}${pathPrefix}/sse/stream`;
    
    document.getElementById('sseEndpoint').textContent = sseUrl;
    
    addMessage(`Connecting to ${sseUrl}...`, 'info');
    
    // Dynamically load and create SSEManager instance
    sseManager = await createSSEManager(sseUrl, {
        reconnect: true,
        reconnectInterval: 5000,
        heartbeatInterval: 2000
    });
    
    sseManager.on('open', function() {
        updateStatus('connected');
        addMessage('Connected successfully!', 'received');
        connectTime = Date.now();
        uptimeInterval = setInterval(updateUptime, 1000);
        reconnectAttempts = 0;
        updateStats();
        document.getElementById('connectBtn').disabled = true;
        document.getElementById('disconnectBtn').disabled = false;
    });
    
    sseManager.on('message', function(data) {
        receivedCount++;
        updateStats();
        const isPrimary = sseManager.getState().isPrimary;
        const prefix = isPrimary ? '' : ' (via coordination)';
        addMessage('Received' + prefix + ': ' + data, 'received');
    });
    
    sseManager.on('error', function(error) {
        addMessage('SSE connection error occurred', 'error');
        console.error('SSE error:', error);
    });
    
    sseManager.on('close', function() {
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
        
        reconnectAttempts++;
        updateStats();
        if (sseManager && sseManager.options.reconnect) {
            addMessage(`Reconnecting in 5 seconds... (Attempt ${reconnectAttempts})`, 'info');
        }
    });
    
    sseManager.on('primary', function() {
        addMessage('This tab is now the PRIMARY connection', 'info');
    });
    
    sseManager.on('secondary', function() {
        addMessage('This tab is SECONDARY (listening via coordination)', 'info');
    });
    
    sseManager.connect();
}

function disconnect() {
    if (sseManager) {
        addMessage('Disconnecting...', 'info');
        sseManager.disconnect();
        sseManager = null;
    }
}

function clearMessages() {
    document.getElementById('messages').innerHTML = '';
    addMessage('Message log cleared', 'info');
}

document.addEventListener('DOMContentLoaded', function() {
    updateStatus('disconnected');
    updateStats();
    updateUptime();
    addMessage('SSE Test Console loaded. Click Connect to start.', 'info');
});

window.addEventListener('beforeunload', function() {
    if (sseManager) {
        sseManager.disconnect();
    }
    if (uptimeInterval) {
        clearInterval(uptimeInterval);
    }
});

// Expose functions to global scope for onclick handlers
window.connect = connect;
window.disconnect = disconnect;
window.clearMessages = clearMessages;
