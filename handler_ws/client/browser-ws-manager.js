/**
 * WebSocket Manager
 * 
 * Provides three connection methods:
 * 1. Shared Worker (primary)
 * 2. IFrame fallback
 * 3. Direct connection with reconnection
 * 
 * All methods ensure only one active connection per browser instance
 * using BroadcastChannel for coordination.
 */
// {{ define "browser-ws-manager" }}
class WebSocketManager {
    constructor(options = {}) {
        this.options = Object.assign({
            debug: false,
            autoConnect: true,
            reconnectOnDisconnect: true,
            maxReconnectAttempts: 10,
            connIdStorageKey: 'ws-conn-id',
            modePrefStorageKey: 'ws-mode-pref',
            coordinationChannel: 'ws-coordination',
            coordinationHeartbeat: 2000, // ms
            assumeDisconnectedAfter: 5000 // ms
        }, options);
        
        // Make connection ID accessible
        this.connectionId = this.getStoredConnectionId();
        this.connectionMode = localStorage.getItem(this.options.modePrefStorageKey) || null;
        this.connectionState = 'disconnected';
        this.reconnectAttempts = 0;
        this.socket = null;
        this.worker = null;
        this.iframe = null;
        this.callbacks = {
            message: [],
            open: [],
            close: [],
            error: []
        };
        
        // Connection coordination properties
        this.broadcastChannel = null;
        this.coordinationCallbacks = {};
        this.isPrimaryConnection = false;
        this.coordinationId = this.generateInstanceId();
        this.lastHeartbeat = null;
        this.heartbeatInterval = null;
        this.electionTimeout = null;

        // Add flag to prevent automatic fallback when mode is explicitly set
        this.explicitModeSet = !!this.connectionMode;

        if (this.options.autoConnect) {
            // Use setTimeout to ensure document.body is available
            setTimeout(() => this.connect(), 0);
        }
    }

    /**
     * Connect to WebSocket using the best available method
     */
    connect() {
        // If this is a secondary connection and coordination is enabled, don't connect
        if (this.broadcastChannel && !this.isPrimaryConnection) {
            this.log('This is a secondary connection, not connecting');
            return;
        }
        
        // If a preferred connection mode is set, use that
        if (this.connectionMode) {
            this.log(`Using preferred connection mode: ${this.connectionMode}`);
            switch (this.connectionMode) {
                case 'worker':
                    if (typeof SharedWorker !== 'undefined') {
                        this.connectViaSharedWorker();
                        return;
                    }
                    this.log('SharedWorker not supported, falling back to default selection');
                    break;
                case 'iframe':
                    this.connectViaIframe();
                    return;
                case 'direct':
                    this.connectDirectly();
                    return;
            }
        }
        
        this.log('Attempting to connect using best available method');
        
        // Try Shared Worker if supported
        if (typeof SharedWorker !== 'undefined') {
            this.log('SharedWorker is supported, trying this method first');
            this.connectViaSharedWorker();
        } else if (this.canUseIframe()) {
            this.log('Using iframe fallback method');
            this.connectViaIframe();
        } else {
            this.log('Using direct WebSocket connection');
            this.connectDirectly();
        }
    }

    /**
     * Set preferred connection mode and save to localStorage
     */
    setPreferredMode(mode) {
        if (['worker', 'iframe', 'direct'].includes(mode)) {
            // If we're changing modes, ensure complete cleanup of previous mode
            if (this.connectionMode !== mode) {
                // Force cleanup of current connection
                this.cleanupCurrentConnection();
                
                // Wait a moment to ensure cleanup completes
                setTimeout(() => {
                    // Clear any global SharedWorker instances that might be cached by the browser
                    if (this.connectionMode === 'worker' && mode !== 'worker') {
                        this.clearSharedWorkers();
                    }
                }, 100);
            }
            
            localStorage.setItem(this.options.modePrefStorageKey, mode);
            this.connectionMode = mode;
            this.explicitModeSet = true;
            this.log(`Set preferred connection mode to: ${mode}`);
            return true;
        }
        return false;
    }

    /**
     * Clear preferred connection mode
     */
    clearPreferredMode() {
        localStorage.removeItem(this.options.modePrefStorageKey);
        this.connectionMode = null;
        this.explicitModeSet = false;
        this.log('Cleared preferred connection mode');
    }

    /**
     * Connect via SharedWorker - Made public for external access
     */
    connectViaSharedWorker() {
        // Clean up any existing connections
        this.cleanupCurrentConnection();
        
        try {
            //this.worker = new SharedWorker('/g/static/js/websocket-worker.js');
            this.worker = new SharedWorker('{{ .WsWorkerRoute }}');
            
            this.worker.port.start();
            
            this.worker.port.addEventListener('message', (event) => {
                this.handleWorkerMessage(event.data);
            });
            
            this.worker.port.postMessage({
                type: 'CONNECT',
                connectionId: this.connectionId
            });
            
            this.connectionMode = 'worker';
            
            this.worker.onerror = (error) => {
                this.log('SharedWorker error', error);
                this.worker = null;
                
                // Only fall back if not explicitly set by user
                if (!this.explicitModeSet) {
                    if (this.canUseIframe()) {
                        this.connectViaIframe();
                    } else {
                        this.connectDirectly();
                    }
                } else {
                    this.log('Not falling back because mode was explicitly set');
                    this.connectionState = 'error';
                    this.triggerCallback('error', { message: 'SharedWorker connection failed' });
                }
            };
        } catch (error) {
            this.log('Failed to initialize SharedWorker', error);
            
            // Only fall back if not explicitly set by user
            if (!this.explicitModeSet) {
                if (this.canUseIframe()) {
                    this.connectViaIframe();
                } else {
                    this.connectDirectly();
                }
            } else {
                this.log('Not falling back because mode was explicitly set');
                this.connectionState = 'error';
                this.triggerCallback('error', { message: 'SharedWorker initialization failed' });
            }
        }
    }

    /**
     * Connect via hidden iframe - Made public for external access
     */
    connectViaIframe() {
        // Clean up any existing connections
        this.cleanupCurrentConnection();
        
        // Ensure document.body is available before creating the iframe
        if (!document.body) {
            this.log('Document body not available yet, delaying iframe creation');
            setTimeout(() => this.connectViaIframe(), 100);
            return;
        }
        
        try {
            this.iframe = document.createElement('iframe');
            this.iframe.style.display = 'none';
            this.iframe.src = '{{ .IframeRoute }}';
            
            // Ensure we have only one message listener
            this.removeMessageListener();
            this.messageListener = (event) => {
                // Only process messages from our iframe
                if (this.iframe && event.source === this.iframe.contentWindow) {
                    this.handleIframeMessage(event.data);
                }
            };
            
            window.addEventListener('message', this.messageListener);
            document.body.appendChild(this.iframe);
            this.connectionMode = 'iframe';
        } catch (error) {
            this.log('Error creating iframe', error);
            // Fall back to direct connection if iframe creation fails
            if (!this.explicitModeSet) {
                this.connectDirectly();
            } else {
                this.connectionState = 'error';
                this.triggerCallback('error', { message: 'Failed to create iframe connection' });
            }
        }
    }

    /**
     * Connect directly to WebSocket server - Made public for external access
     */
    connectDirectly() {
        // Clean up any existing connections
        this.cleanupCurrentConnection();
        
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${protocol}//${window.location.host}{{ .Route }}?connid=${this.connectionId}`;
        
        this.socket = new WebSocket(url);
        this.connectionMode = 'direct';
        
        this.socket.onopen = () => {
            this.log('Direct WebSocket connected');
            this.connectionState = 'connected';
            this.reconnectAttempts = 0;
            this.triggerCallback('open');
        };
        
        this.socket.onclose = () => {
            this.log('Direct WebSocket closed');
            this.connectionState = 'disconnected';
            this.triggerCallback('close');
            
            if (this.options.reconnectOnDisconnect && !this.explicitModeSet) {
                this.attemptReconnect();
            }
        };
        
        this.socket.onerror = (error) => {
            this.log('Direct WebSocket error', error);
            this.triggerCallback('error', error);
        };
        
        this.socket.onmessage = (event) => {
            this.log('Direct WebSocket message received', event.data);
            this.triggerCallback('message', event.data);
        };
    }

    /**
     * Clean up the current connection before establishing a new one
     */
    cleanupCurrentConnection() {
        // Set state to disconnected
        this.connectionState = 'disconnected';
        
        // Clean up worker
        if (this.worker) {
            try {
                // Send disconnect message to the worker
                this.worker.port.postMessage({ type: 'DISCONNECT' });
                // Close the port connection
                this.worker.port.close();
                
                // Terminate the worker if possible (not all browsers support this)
                if (typeof this.worker.terminate === 'function') {
                    this.worker.terminate();
                }
                
                // Force worker to be garbage collected
                this.log('Worker connection terminated');
            } catch (e) {
                this.log('Error cleaning up worker', e);
            }
            this.worker = null;
        }
        
        // Clean up iframe
        if (this.iframe) {
            try {
                // Remove event listeners first to prevent any race conditions
                this.removeMessageListener();
                
                // Remove iframe from DOM
                if (document.body.contains(this.iframe)) {
                    document.body.removeChild(this.iframe);
                }
            } catch (e) {
                this.log('Error cleaning up iframe', e);
            }
            this.iframe = null;
        }
        
        // Clean up direct connection
        if (this.socket) {
            try {
                this.socket.close();
            } catch (e) {
                this.log('Error cleaning up direct socket', e);
            }
            this.socket = null;
        }
        
        // Kill any pending reconnect attempts
        if (this.reconnectTimeout) {
            clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = null;
        }
    }

    /**
     * Remove message listener for iframe communication
     */
    removeMessageListener() {
        if (this.messageListener) {
            window.removeEventListener('message', this.messageListener);
            this.messageListener = null;
        }
    }

    /**
     * Send message through the active connection
     * or relay through the primary connection if this is a secondary tab
     */
    send(message) {
        if (typeof message !== 'string') {
            message = JSON.stringify(message);
        }
        
        // If this is a secondary connection and coordination is enabled,
        // send through the broadcast channel
        if (this.broadcastChannel && !this.isPrimaryConnection) {
            this.log('Sending message via coordination channel (secondary tab)');
            
            // Check if we already have a pending request for this message
            if (!this.pendingSendRequests) {
                this.pendingSendRequests = new Map();
            }
            
            const messageKey = message;
            const now = Date.now();
            
            // Check if we've recently sent this exact message
            if (this.pendingSendRequests.has(messageKey)) {
                const lastRequest = this.pendingSendRequests.get(messageKey);
                if (now - lastRequest.timestamp < 1000) { // Within 1 second
                    this.log('Duplicate send request detected, ignoring');
                    return true;
                }
            }
            
            // Generate a unique request ID to track responses
            const requestId = `${this.coordinationId}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
            
            // Store this request to prevent duplicates
            this.pendingSendRequests.set(messageKey, {
                requestId: requestId,
                timestamp: now
            });
            
            // Clean up old requests after 5 seconds
            setTimeout(() => {
                if (this.pendingSendRequests) {
                    this.pendingSendRequests.delete(messageKey);
                }
            }, 5000);
            
            // Send only once, with proper identification
            this.broadcastChannel.postMessage({
                type: 'send-request',
                requestId: requestId,
                senderId: this.coordinationId,  // Add sender ID for better filtering
                id: this.coordinationId,
                message: message,
                timestamp: now
            });
            return true; // Return early, don't try to send directly
        }
        
        // For primary connection or when coordination is not active
        if (this.connectionState !== 'connected') {
            this.log('Cannot send message, not connected');
            return false;
        }
        
        switch (this.connectionMode) {
            case 'worker':
                this.worker.port.postMessage({
                    type: 'SEND',
                    data: message
                });
                break;
                
            case 'iframe':
                if (this.iframe && this.iframe.contentWindow) {
                    this.iframe.contentWindow.postMessage({
                        type: 'WS_SEND',
                        data: message
                    }, '*');
                } else {
                    return false;
                }
                break;
                
            case 'direct':
                if (this.socket && this.socket.readyState === WebSocket.OPEN) {
                    this.socket.send(message);
                } else {
                    return false;
                }
                break;
                
            default:
                return false;
        }
        
        return true;
    }

    /**
     * Handle messages from the SharedWorker
     */
    handleWorkerMessage(data) {
        switch (data.type) {
            case 'CONNECTED':
                this.log('WebSocket connected via SharedWorker');
                this.connectionState = 'connected';
                this.triggerCallback('open');
                break;
                
            case 'DISCONNECTED':
                this.log('WebSocket disconnected via SharedWorker');
                this.connectionState = 'disconnected';
                this.triggerCallback('close');
                
                if (this.options.reconnectOnDisconnect) {
                    this.worker.port.postMessage({
                        type: 'RECONNECT',
                        connectionId: this.connectionId
                    });
                }
                break;
                
            case 'MESSAGE':
                this.log('WebSocket message received via SharedWorker', data.data);
                // Process the message directly in primary tab
                this.triggerCallback('message', data.data);
                
                // Only relay to other tabs if we're the primary
                if (this.broadcastChannel && this.isPrimaryConnection) {
                    this.broadcastChannel.postMessage({
                        type: 'message-relay',
                        id: this.coordinationId,
                        message: data.data,
                        timestamp: Date.now()
                    });
                }
                break;
                
            case 'ERROR':
                this.log('WebSocket error via SharedWorker', data.error);
                this.triggerCallback('error', data.error);
                break;
                
            default:
                this.log('Unknown message from SharedWorker', data);
        }
    }

    /**
     * Handle messages from the iframe
     */
    handleIframeMessage(data) {
        switch (data.type) {
            case 'WS_CONNECTED':
                this.log('WebSocket connected via iframe');
                this.connectionState = 'connected';
                this.triggerCallback('open');
                break;
                
            case 'WS_CLOSED':
                this.log('WebSocket disconnected via iframe');
                this.connectionState = 'disconnected';
                this.triggerCallback('close');
                break;
                
            case 'WS_MESSAGE':
                this.log('WebSocket message received via iframe', data.data);
                // Process the message directly in primary tab
                this.triggerCallback('message', data.data);
                
                // Only relay to other tabs if we're the primary
                if (this.broadcastChannel && this.isPrimaryConnection) {
                    this.broadcastChannel.postMessage({
                        type: 'message-relay',
                        id: this.coordinationId,
                        message: data.data,
                        timestamp: Date.now()
                    });
                }
                break;
                
            case 'WS_ERROR':
                this.log('WebSocket error via iframe');
                this.triggerCallback('error', { message: 'WebSocket error in iframe' });
                break;
                
            case 'WS_RECONNECTING':
                this.log(`Reconnecting via iframe (attempt ${data.attempt} after ${data.delay}ms)`);
                break;
                
            case 'WS_MAX_RECONNECT_ATTEMPTS':
                this.log('Maximum reconnection attempts reached in iframe');
                break;
                
            default:
                this.log('Unknown message from iframe', data);
        }
    }

    /**
     * Handle coordination messages
     */
    handleCoordinationMessage(data) {
        try {
            if (!data || !data.type) {
                this.log('Received invalid coordination message', data);
                return;
            }
            
            // Filter out our own messages for certain types
            if (['send-request', 'send-response'].includes(data.type) && data.senderId === this.coordinationId) {
                // Ignore our own send requests and responses
                return;
            }
            
            switch (data.type) {
                case 'presence':
                case 'heartbeat':
                    // Update last seen timestamp for this instance
                    this.lastHeartbeat = Date.now();
                    
                    // Add/update this tab in known tabs list
                    if (this.knownTabs && data.id !== this.coordinationId) {
                        const existing = this.knownTabs.has(data.id);
                        this.knownTabs.set(data.id, {
                            id: data.id,
                            isPrimary: data.isPrimary,
                            lastSeen: Date.now()
                        });
                        
                        // Broadcast updated tabs list if this is a new tab
                        if (!existing) {
                            this.broadcastTabsUpdate();
                        }
                    }
                    break;
                    
                case 'election':
                    // If our ID is "greater", we object and start a new election
                    if (this.coordinationId > data.id) {
                        this.broadcastChannel.postMessage({
                            type: 'election-objection',
                            id: this.coordinationId,
                            timestamp: Date.now()
                        });
                        
                        // Start our own election
                        setTimeout(() => this.initiateElection(), 100);
                    }
                    break;
                    
                case 'election-objection':
                    // Someone objected to our election, cancel our timeout
                    if (this.electionTimeout) {
                        clearTimeout(this.electionTimeout);
                        this.electionTimeout = null;
                    }
                    break;
                    
                case 'primary-elected':
                    // Someone else became primary
                    if (data.id !== this.coordinationId) {
                        this.becomeSecondary();
                    }
                    break;
                    
                case 'primary-disconnected':
                    // Primary connection closed, initiate new election
                    if (data.id !== this.coordinationId) {
                        setTimeout(() => this.initiateElection(), 100 + Math.random() * 400);
                    }
                    break;
                    
                case 'message-relay':
                    // Relay messages between tabs - only process if this is a secondary connection
                    if (data.id !== this.coordinationId) {
                        // Check if we're a secondary tab before processing the relay
                        if (!this.isPrimaryConnection) {
                            this.log('Received relayed message in secondary tab');
                            this.triggerCallback('message', data.message);
                        } else {
                            this.log('Ignoring relayed message in primary tab (already received directly)');
                        }
                    }
                    break;
                    
                case 'send-request':
                    // Handle send requests from secondary tabs when we're primary
                    // Check both that we're primary AND this isn't our own message
                    if (this.isPrimaryConnection && data.id !== this.coordinationId && data.senderId !== this.coordinationId) {
                        this.log('Received send request from secondary tab');
                        
                        // Use the request ID for deduplication
                        const messageId = data.requestId;
                        
                        if (!this.recentlySentMessages) {
                            this.recentlySentMessages = new Set();
                        }
                        
                        if (this.recentlySentMessages.has(messageId)) {
                            this.log('Duplicate send request detected, ignoring');
                            // Still send response to avoid hanging the secondary tab
                            this.broadcastChannel.postMessage({
                                type: 'send-response',
                                requestId: data.requestId,
                                targetId: data.senderId,  // Target specific sender
                                senderId: this.coordinationId,
                                success: false,
                                duplicate: true,
                                timestamp: Date.now(),
                                id: this.coordinationId
                            });
                            return;
                        }
                        
                        // Add to recent messages set
                        this.recentlySentMessages.add(messageId);
                        
                        // Clean up old message IDs after 10 seconds
                        setTimeout(() => {
                            if (this.recentlySentMessages) {
                                this.recentlySentMessages.delete(messageId);
                            }
                        }, 10000);
                        
                        // Forward the message to the actual WebSocket
                        let success = false;
                        
                        // Only send if we have an active connection
                        if (this.connectionState !== 'connected') {
                            this.log('Cannot forward message, primary not connected');
                            this.broadcastChannel.postMessage({
                                type: 'send-response',
                                requestId: data.requestId,
                                targetId: data.senderId,
                                senderId: this.coordinationId,
                                success: false,
                                error: 'not_connected',
                                timestamp: Date.now(),
                                id: this.coordinationId
                            });
                            return;
                        }
                        
                        // Send directly to the actual socket
                        switch (this.connectionMode) {
                            case 'worker':
                                if (this.worker) {
                                    this.worker.port.postMessage({
                                        type: 'SEND',
                                        data: data.message
                                    });
                                    success = true;
                                }
                                break;
                                
                            case 'iframe':
                                if (this.iframe && this.iframe.contentWindow) {
                                    this.iframe.contentWindow.postMessage({
                                        type: 'WS_SEND',
                                        data: data.message
                                    }, '*');
                                    success = true;
                                }
                                break;
                                
                            case 'direct':
                                if (this.socket && this.socket.readyState === WebSocket.OPEN) {
                                    this.socket.send(data.message);
                                    success = true;
                                }
                                break;
                        }
                        
                        // Send response targeted to the specific sender
                        this.broadcastChannel.postMessage({
                            type: 'send-response',
                            requestId: data.requestId,
                            targetId: data.senderId,
                            senderId: this.coordinationId,
                            success: success,
                            timestamp: Date.now(),
                            id: this.coordinationId
                        });
                    }
                    break;
                
                case 'send-response':
                    // Only process responses targeted to us
                    if (!this.isPrimaryConnection && data.targetId === this.coordinationId) {
                        if (data.duplicate) {
                            this.log('Send request was duplicate');
                        } else if (data.error === 'not_connected') {
                            this.log('Send request failed: primary not connected');
                        } else {
                            this.log(`Send request ${data.success ? 'succeeded' : 'failed'}`);
                        }
                    }
                    break;
                    
                case 'tabs-status':
                    // Update with remote tab's view of all tabs
                    if (this.knownTabs && data.id !== this.coordinationId && data.tabs) {
                        let tabsChanged = false;
                        
                        data.tabs.forEach(tab => {
                            if (tab.id !== this.coordinationId) { // Don't overwrite our own entry
                                const existing = this.knownTabs.has(tab.id);
                                const currentTab = existing ? this.knownTabs.get(tab.id) : null;
                                
                                if (!existing || (currentTab && currentTab.isPrimary !== tab.isPrimary)) {
                                    this.knownTabs.set(tab.id, tab);
                                    tabsChanged = true;
                                }
                            }
                        });
                        
                        if (tabsChanged) {
                            this.triggerCoordinationCallback('tabs-updated', Array.from(this.knownTabs.values()));
                        }
                    }
                    break;
                    
                default:
                    this.log(`Unknown coordination message type: ${data.type}`, data);
                    break;
            }
        } catch (error) {
            this.log('Error handling coordination message', error);
        }
    }

    /**
     * Attempt to reconnect with exponential backoff
     */
    attemptReconnect() {
        if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
            this.log('Maximum reconnection attempts reached');
            return;
        }
        
        const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
        this.reconnectAttempts++;
        
        this.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);
        
        setTimeout(() => {
            if (this.connectionState === 'disconnected') {
                this.connectDirectly();
            }
        }, delay);
    }

    /**
     * Register event callbacks
     */
    on(event, callback) {
        if (this.callbacks[event]) {
            this.callbacks[event].push(callback);
        }
        return this;
    }

    /**
     * Trigger callbacks for an event
     * Simplified deduplication for message events
     */
    triggerCallback(event, data) {
        if (event === 'message') {
            // Simplified deduplication - only check for exact same message within 1 second
            const messageKey = typeof data === 'string' ? data : JSON.stringify(data);
            
            if (!this.recentMessages) {
                this.recentMessages = new Map();
            }
            
            const now = Date.now();
            const lastSeen = this.recentMessages.get(messageKey);
            
            // Only check for duplicates within 1 second window
            if (lastSeen && now - lastSeen < 1000) {
                this.log('Duplicate message detected within 1 second, ignoring');
                return;
            }
            
            // Store the message timestamp
            this.recentMessages.set(messageKey, now);
            
            // Clean up old messages every 10 seconds
            if (!this.messageCleanupInterval) {
                this.messageCleanupInterval = setInterval(() => {
                    const expiry = Date.now() - 5000; // Keep messages for 5 seconds
                    if (this.recentMessages) {
                        this.recentMessages.forEach((timestamp, msg) => {
                            if (timestamp < expiry) {
                                this.recentMessages.delete(msg);
                            }
                        });
                    }
                }, 10000);
            }
        }
        
        // Existing callback triggering code
        if (this.callbacks[event]) {
            this.callbacks[event].forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error('Error in WebSocket callback', error);
                }
            });
        }
    }

    /**
     * Check if iframe method can be used
     */
    canUseIframe() {
        return true; // Always possible as fallback
    }

    /**
     * Get or generate connection ID from storage
     * Returns the current connection ID
     */
    getStoredConnectionId() {
        let id = localStorage.getItem(this.options.connIdStorageKey);
        if (!id) {
            id = this.generateConnectionId();
            localStorage.setItem(this.options.connIdStorageKey, id);
        }
        return id;
    }

    /**
     * Reset the connection ID to a new value
     * Useful when you want to force a new connection
     */
    resetConnectionId() {
        this.connectionId = this.generateConnectionId();
        localStorage.setItem(this.options.connIdStorageKey, this.connectionId);
        return this.connectionId;
    }

    /**
     * Disconnect WebSocket
     */
    disconnect(suppressEvents = false) {
        this.connectionState = 'disconnected';
        
        switch (this.connectionMode) {
            case 'worker':
                if (this.worker) {
                    try {
                        this.worker.port.postMessage({ type: 'DISCONNECT' });
                        this.worker.port.close();
                    } catch (e) {
                        this.log('Error closing worker', e);
                    }
                    this.worker = null;
                }
                break;
                
            case 'iframe':
                if (this.iframe) {
                    this.removeMessageListener();
                    try {
                        document.body.removeChild(this.iframe);
                    } catch (e) {
                        this.log('Error removing iframe', e);
                    }
                    this.iframe = null;
                }
                break;
                
            case 'direct':
                if (this.socket) {
                    try {
                        this.socket.close();
                    } catch (e) {
                        this.log('Error closing socket', e);
                    }
                    this.socket = null;
                }
                break;
        }
        
        // If this is the primary connection disconnecting and coordination is enabled,
        // notify other tabs unless suppressed (for coordination-managed disconnects)
        if (!suppressEvents && this.isPrimaryConnection && this.broadcastChannel) {
            this.broadcastChannel.postMessage({
                type: 'primary-disconnected',
                id: this.coordinationId,
                timestamp: Date.now()
            });
        }
        
        this.triggerCallback('close');
    }

    /**
     * Completely reset the connection
     */
    resetConnection() {
        this.disconnect();
        this.clearPreferredMode();
        this.connectionId = this.resetConnectionId();
        return this.connect();
    }

    /**
     * Log message if debug is enabled
     */
    log(message, data) {
        if (this.options.debug) {
            if (data) {
                console.log(`[WebSocketManager] ${message}`, data);
            } else {
                console.log(`[WebSocketManager] ${message}`);
            }
        }
    }

    /**
     * Try to clear any existing SharedWorkers
     * Note: This is a best-effort approach as browsers limit what we can do with workers
     */
    clearSharedWorkers() {
        this.log('Attempting to clear SharedWorkers');
        
        // Create a temporary worker to send a global shutdown message
        try {
            const tempWorker = new SharedWorker('{{ .WsWorkerRoute }}');
            tempWorker.port.start();
            tempWorker.port.postMessage({ type: 'GLOBAL_SHUTDOWN' });
            
            // Close the port after sending the message
            setTimeout(() => {
                try {
                    tempWorker.port.close();
                } catch (e) {
                    // Ignore errors on cleanup
                }
            }, 100);
        } catch (e) {
            this.log('Error creating temporary worker for cleanup', e);
        }
        
        // Additional fallback - notify the server about mode change
        // to help it clean up orphaned connections
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const url = `${protocol}//${window.location.host}{{ .Route }}?connid=${this.connectionId}&cleanup=1`;
            
            const cleanupSocket = new WebSocket(url);
            cleanupSocket.onopen = () => {
                cleanupSocket.send(JSON.stringify({
                    type: 'mode_change',
                    previousMode: 'worker',
                    newMode: this.connectionMode
                }));
                
                // Close immediately after sending
                setTimeout(() => cleanupSocket.close(), 50);
            };
        } catch (e) {
            this.log('Error sending cleanup notification', e);
        }
    }

    /**
     * Initialize connection coordination using BroadcastChannel
     */
    initConnectionCoordination() {
        // Check if BroadcastChannel is supported
        if (typeof BroadcastChannel === 'undefined') {
            this.log('BroadcastChannel not supported, coordination disabled');
            return false;
        }
        
        try {
            // Create or get the broadcast channel
            this.broadcastChannel = new BroadcastChannel(this.options.coordinationChannel);
            
            // Use arrow function to preserve 'this' context
            this.broadcastChannel.onmessage = (event) => {
                // Add global message deduplication
                if (!this.recentCoordinationMessages) {
                    this.recentCoordinationMessages = new Map();
                }
                
                const messageKey = JSON.stringify(event.data);
                const now = Date.now();
                
                // Check if we've seen this exact message recently
                const lastSeen = this.recentCoordinationMessages.get(messageKey);
                if (lastSeen && now - lastSeen < 100) { // Within 100ms
                    // Ignore duplicate coordination message
                    return;
                }
                
                // Store this message
                this.recentCoordinationMessages.set(messageKey, now);
                
                // Clean up old messages periodically
                if (!this.coordinationCleanupInterval) {
                    this.coordinationCleanupInterval = setInterval(() => {
                        const expiry = Date.now() - 1000; // Keep for 1 second
                        this.recentCoordinationMessages.forEach((timestamp, msg) => {
                            if (timestamp < expiry) {
                                this.recentCoordinationMessages.delete(msg);
                            }
                        });
                    }, 5000);
                }
                
                this.handleCoordinationMessage(event.data);
            };
            
            // Start the coordination process
            this.startCoordination();
            
            // Trigger coordination enabled event
            this.triggerCoordinationCallback('coordination-enabled');
            
            return true;
        } catch (e) {
            this.log('Error initializing coordination', e);
            return false;
        }
    }
    
    /**
     * Start the coordination process
     */
    startCoordination() {
        // Clear any existing intervals/timeouts
        this.clearCoordinationTimers();
        
        // Send presence announcement
        this.broadcastPresence();
        
        // Start heartbeat interval
        this.heartbeatInterval = setInterval(() => {
            this.broadcastHeartbeat();
        }, this.options.coordinationHeartbeat);
        
        // Initiate election
        this.initiateElection();
        
        // Track all known tabs
        this.knownTabs = new Map();
        this.knownTabs.set(this.coordinationId, {
            id: this.coordinationId,
            isPrimary: this.isPrimaryConnection,
            lastSeen: Date.now()
        });
        
        // Add periodic tab cleanup and status update
        setInterval(() => {
            this.cleanupStaleTabsAndUpdateStatus();
        }, this.options.coordinationHeartbeat * 2);
        
        // Handle page visibility changes
        document.addEventListener('visibilitychange', () => {
            if (document.visibilityState === 'visible') {
                // Re-announce presence when page becomes visible
                this.broadcastPresence();
                
                // Re-initiate election if we were primary
                if (this.isPrimaryConnection) {
                    this.initiateElection();
                }
            }
        });
        
        // Handle before unload to notify other tabs
        window.addEventListener('beforeunload', () => {
            if (this.isPrimaryConnection) {
                this.broadcastChannel.postMessage({
                    type: 'primary-disconnected',
                    id: this.coordinationId,
                    timestamp: Date.now()
                });
            }
        });
    }
    
    /**
     * Clear coordination timers
     */
    clearCoordinationTimers() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
        
        if (this.electionTimeout) {
            clearTimeout(this.electionTimeout);
            this.electionTimeout = null;
        }
    }
    
    /**
     * Broadcast presence to other tabs
     */
    broadcastPresence() {
        if (!this.broadcastChannel) return;
        
        this.broadcastChannel.postMessage({
            type: 'presence',
            id: this.coordinationId,
            timestamp: Date.now(),
            isPrimary: this.isPrimaryConnection,
            connectionState: this.connectionState
        });
        
        // Update this tab in known tabs
        if (this.knownTabs) {
            this.knownTabs.set(this.coordinationId, {
                id: this.coordinationId,
                isPrimary: this.isPrimaryConnection,
                lastSeen: Date.now()
            });
            
            // Notify about tabs update
            this.broadcastTabsUpdate();
        }
    }
    
    /**
     * Clean up stale tabs and update status
     */
    cleanupStaleTabsAndUpdateStatus() {
        if (!this.knownTabs) return;
        
        const now = Date.now();
        const cutoff = now - (this.options.assumeDisconnectedAfter * 2);
        
        // Remove stale tabs
        let tabsChanged = false;
        this.knownTabs.forEach((tab, id) => {
            if (tab.lastSeen < cutoff) {
                this.knownTabs.delete(id);
                tabsChanged = true;
            }
        });
        
        if (tabsChanged) {
            this.broadcastTabsUpdate();
        }
    }
    
    /**
     * Broadcast tabs update
     */
    broadcastTabsUpdate() {
        if (!this.knownTabs) return;
        
        const tabsList = Array.from(this.knownTabs.values());
        
        // Notify local listeners
        this.triggerCoordinationCallback('tabs-updated', tabsList);
        
        // Broadcast tabs status to other tabs
        if (this.broadcastChannel) {
            this.broadcastChannel.postMessage({
                type: 'tabs-status',
                id: this.coordinationId,
                timestamp: Date.now(),
                tabs: tabsList
            });
        }
    }
    
    /**
     * Broadcast heartbeat to other tabs
     */
    broadcastHeartbeat() {
        if (!this.broadcastChannel) return;
        
        this.broadcastChannel.postMessage({
            type: 'heartbeat',
            id: this.coordinationId,
            timestamp: Date.now(),
            isPrimary: this.isPrimaryConnection,
            connectionState: this.connectionState
        });
    }
    
    /**
     * Initiate election to determine primary connection
     */
    initiateElection() {
        if (!this.broadcastChannel) return;
        
        // Send election message
        this.broadcastChannel.postMessage({
            type: 'election',
            id: this.coordinationId,
            timestamp: Date.now()
        });
        
        // Set timeout to declare self as primary if no one objects
        this.electionTimeout = setTimeout(() => {
            this.becomePrimary();
        }, 500);
    }
    
    /**
     * Become the primary connection
     */
    becomePrimary() {
        if (this.isPrimaryConnection) return;
        
        this.log('Becoming primary connection');
        this.isPrimaryConnection = true;
        
        // Announce primary status
        this.broadcastChannel.postMessage({
            type: 'primary-elected',
            id: this.coordinationId,
            timestamp: Date.now()
        });
        
        // Ensure connection is active
        if (this.connectionState !== 'connected') {
            this.connect();
        }
        
        // Update our entry in known tabs
        if (this.knownTabs) {
            this.knownTabs.set(this.coordinationId, {
                id: this.coordinationId,
                isPrimary: true,
                lastSeen: Date.now()
            });
            this.broadcastTabsUpdate();
        }
        
        // Trigger became-primary event
        this.triggerCoordinationCallback('became-primary');
    }
    
    /**
     * Become a secondary connection
     */
    becomeSecondary() {
        if (!this.isPrimaryConnection) return;
        
        this.log('Becoming secondary connection');
        this.isPrimaryConnection = false;
        
        // Disconnect if connected (primary will handle the connection)
        if (this.connectionState === 'connected') {
            this.disconnect(true);
        }
        
        // Update our entry in known tabs
        if (this.knownTabs) {
            this.knownTabs.set(this.coordinationId, {
                id: this.coordinationId,
                isPrimary: false,
                lastSeen: Date.now()
            });
            this.broadcastTabsUpdate();
        }
        
        // Trigger became-secondary event
        this.triggerCoordinationCallback('became-secondary');
    }
    
    /**
     * Register coordination event callbacks
     */
    onCoordinationEvent(event, callback) {
        if (!this.coordinationCallbacks[event]) {
            this.coordinationCallbacks[event] = [];
        }
        this.coordinationCallbacks[event].push(callback);
        return this;
    }
    
    /**
     * Trigger coordination callbacks for an event
     */
    triggerCoordinationCallback(event, data) {
        if (this.coordinationCallbacks[event]) {
            this.coordinationCallbacks[event].forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error('Error in coordination callback', error);
                }
            });
        }
    }
    
    /**
     * Generate a unique instance ID for coordination
     */
    generateInstanceId() {
        return Date.now().toString() + Math.random().toString(36).substring(2, 9);
    }
    
    /**
     * Generate a connection ID
     */
    generateConnectionId() {
        return Date.now().toString() + Math.random().toString(36).substring(2, 9);
    }
    
    /**
     * Clean up and dispose all resources
     */
    dispose() {
        this.disconnect();
        
        // Clean up intervals
        if (this.messageCleanupInterval) {
            clearInterval(this.messageCleanupInterval);
            this.messageCleanupInterval = null;
        }
        
        if (this.coordinationCleanupInterval) {
            clearInterval(this.coordinationCleanupInterval);
            this.coordinationCleanupInterval = null;
        }
        
        // Clear message caches
        if (this.recentMessages) {
            this.recentMessages.clear();
        }
        
        if (this.recentlySentMessages) {
            this.recentlySentMessages.clear();
        }
        
        if (this.recentCoordinationMessages) {
            this.recentCoordinationMessages.clear();
        }
        
        // Clear pending send requests
        if (this.pendingSendRequests) {
            this.pendingSendRequests.clear();
        }
        
        if (this.broadcastChannel) {
            this.clearCoordinationTimers();
            
            if (this.isPrimaryConnection) {
                this.broadcastChannel.postMessage({
                    type: 'primary-disconnected',
                    id: this.coordinationId,
                    timestamp: Date.now()
                });
            }
            
            this.broadcastChannel.close();
            this.broadcastChannel = null;
        }
    }
    
    /**
     * Get all known tabs
     */
    getKnownTabs() {
        return this.knownTabs ? Array.from(this.knownTabs.values()) : [];
    }
    
    /**
     * Check if this tab is the primary connection
     */
    isPrimary() {
        return this.isPrimaryConnection;
    }
}

// Create global instance only after DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.wsManager = new WebSocketManager({ debug: true });
        
        // Setup coordination once DOM is ready
        if (window.wsManager.initConnectionCoordination) {
            window.wsManager.initConnectionCoordination();
        }
    });
} else {
    // DOM is already ready
    window.wsManager = new WebSocketManager({ debug: true });
    
    // Setup coordination immediately
    if (window.wsManager.initConnectionCoordination) {
        window.wsManager.initConnectionCoordination();
    }
}
// {{ end }}
