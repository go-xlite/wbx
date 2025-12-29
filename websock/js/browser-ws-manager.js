
/**
 * WebSocket Manager
 * 
 * Provides two connection methods:
 * 1. Shared Worker (desktop multi-tab coordination)
 * 2. Direct connection (mobile/per-tab)
 * 
 * SharedWorker mode uses BroadcastChannel for tab coordination.
 * Direct mode creates independent connections per tab.
 */


// Template configuration - replace these values with simple string replacement after minification
const WS_CONFIG = {
    workerRoute: '__WS_WORKER_ROUTE__',
    wsRoute: '__WS_ROUTE__'
};

// Connection states
const STATE_DISCONNECTED = 1;
const STATE_CONNECTING = 2;
const STATE_CONNECTED = 4;
const STATE_ERROR = 8;

// Connection modes
const MODE_WORKER = 1;
const MODE_DIRECT = 4;

// Event types
const EVENT_MESSAGE = 1;
const EVENT_OPEN = 2;
const EVENT_ERROR = 4;
const EVENT_CLOSE = 8;

// Coordination message types
const COORD_SEND_REQUEST = 1;
const COORD_SEND_RESPONSE = 2;
const COORD_MESSAGE_RELAY = 4;
const COORD_PRESENCE = 8;
const COORD_HEARTBEAT = 16;
const COORD_ELECTION = 32;
const COORD_ELECTION_OBJECTION = 64;
const COORD_PRIMARY_ELECTED = 128;
const COORD_PRIMARY_DISCONNECTED = 256;
const COORD_TABS_STATUS = 512;

// Worker message types
const WORKER_CONNECT = 1;
const WORKER_DISCONNECT = 2;
const WORKER_SEND = 4;
const WORKER_CONNECTED = 8;
const WORKER_DISCONNECTED = 16;
const WORKER_MESSAGE = 32;
const WORKER_ERROR = 64;
const WORKER_RECONNECT = 128;
const WORKER_GLOBAL_SHUTDOWN = 256;

// Coordination callback events
const COORD_CB_ENABLED = 1;
const COORD_CB_BECAME_PRIMARY = 2;
const COORD_CB_BECAME_SECONDARY = 4;
const COORD_CB_TABS_UPDATED = 8;

// Other constants
const MSG_TYPE_MODE_CHANGE = 1;
const ERROR_NOT_CONNECTED = 2;


class WebSocketManager {
    // Private fields
    #options;
    #reconnectAttempts;
    #socket;
    #worker;
    #callbacks;
    #coordinationCallbacks;
    #isPrimaryConnection;
    #coordinationId;
    #heartbeatInterval;
    #electionTimeout;
    #explicitModeSet;
    #messageListener;
    #recentMessages;
    #recentlySentMessages;
    #recentCoordinationMessages;
    #pendingSendRequests;
    #knownTabs;
    #messageCleanupInterval;
    #coordinationCleanupInterval;
    #reconnectTimeout;

    constructor(options = {}) {
        this.#options = Object.assign({
            debug: false,
            autoConnect: true,
            reconnectOnDisconnect: true,
            maxReconnectAttempts: 10,
            connIdStorageKey: 'ws-conn-id',
            modePrefStorageKey: 'ws-mode-pref',
            coordinationChannel: 'ws-coordination',
            coordinationHeartbeat: 2000, // ms
            assumeDisconnectedAfter: 10000 // ms - increased for mobile throttling tolerance
        }, options);
        
        // Public properties
        this.connectionId = this.#getStoredConnectionId();
        this.connectionMode = localStorage.getItem(this.#options.modePrefStorageKey) || null;
        this.connectionState = STATE_DISCONNECTED;
        this.broadcastChannel = null;
        
        // Private properties
        this.#reconnectAttempts = 0;
        this.#socket = null;
        this.#worker = null;
        this.#callbacks = {
            [EVENT_MESSAGE]: [],
            [EVENT_OPEN]: [],
            [EVENT_CLOSE]: [],
            [EVENT_ERROR]: []
        };
        this.#coordinationCallbacks = {};
        this.#isPrimaryConnection = false;
        this.#coordinationId = this.#generateInstanceId();
        this.#heartbeatInterval = null;
        this.#electionTimeout = null;
        this.#explicitModeSet = !!this.connectionMode;

        if (this.#options.autoConnect) {
            setTimeout(() => this.connect(), 0);
        }
    }

    /**
     * Connect to WebSocket using the best available method
     */
    connect() {
        // If this is a secondary connection and coordination is enabled, don't connect
        if (this.broadcastChannel && !this.#isPrimaryConnection) {
            console.log('[WS] This is a secondary connection, not connecting');
            return;
        }
        
        // If a preferred connection mode is set, use that
        if (this.connectionMode) {
            console.log(`[WS] Using preferred connection mode: ${this.connectionMode}`);
            switch (this.connectionMode) {
                case MODE_WORKER:
                    if (typeof SharedWorker !== 'undefined') {
                        this.connectViaSharedWorker();
                        return;
                    }
                    console.log('[WS] SharedWorker not supported, falling back to default selection');
                    break;
                case MODE_DIRECT:
                    this.connectDirectly();
                    return;
            }
        }
        
        console.log('[WS] Attempting to connect using best available method');
        
        // Try Shared Worker if supported
        if (typeof SharedWorker !== 'undefined') {
            console.log('[WS] SharedWorker is supported, trying this method first');
            this.connectViaSharedWorker();
        } else {
            console.log('[WS] Using direct WebSocket connection');
            this.connectDirectly();
        }
    }

    /**
     * Set preferred connection mode and save to localStorage
     */
    setPreferredMode(mode) {
        if ([MODE_WORKER, MODE_DIRECT].includes(mode)) {
            // If we're changing modes, ensure complete cleanup of previous mode
            if (this.connectionMode !== mode) {
                // Force cleanup of current connection
                this.#cleanupCurrentConnection();
            }
            
            localStorage.setItem(this.#options.modePrefStorageKey, mode);
            this.connectionMode = mode;
            this.#explicitModeSet = true;
            console.log(`[WS] Set preferred connection mode to: ${mode}`);
            return true;
        }
        return false;
    }

    /**
     * Clear preferred connection mode
     */
    clearPreferredMode() {
        localStorage.removeItem(this.#options.modePrefStorageKey);
        this.connectionMode = null;
        this.#explicitModeSet = false;
        console.log('[WS] Cleared preferred connection mode');
    }

    /**
     * Connect via SharedWorker - Made public for external access
     */
    connectViaSharedWorker() {
        // Clean up any existing connections
        this.#cleanupCurrentConnection();
        
        // Re-initialize BroadcastChannel for coordination only for SharedWorker mode
        if (!this.broadcastChannel) {
            this.initConnectionCoordination();
        }
        
        try {
            //this.#worker = new SharedWorker('/g/static/js/websocket-worker.js');
            this.#worker = new SharedWorker(WS_CONFIG.workerRoute);
            
            this.#worker.port.start();
            
            this.#worker.port.addEventListener('message', (event) => {
                this.#handleWorkerMessage(event.data);
            });
            
            this.#worker.port.postMessage({
                type: WORKER_CONNECT,
                connectionId: this.connectionId
            });
            
            this.connectionMode = MODE_WORKER;
            
            this.#worker.onerror = (error) => {
                this.#log('SharedWorker error', error);
                this.#worker = null;
                
                // Only fall back if not explicitly set by user
                if (!this.#explicitModeSet) {
                    this.connectDirectly();
                } else {
                    this.#log('Not falling back because mode was explicitly set');
                    this.connectionState = STATE_ERROR;
                    this.#triggerCallback(EVENT_ERROR, { message: 'SharedWorker connection failed' });
                }
            };
        } catch (error) {
            this.#log('Failed to initialize SharedWorker', error);
            
            // Only fall back if not explicitly set by user
            if (!this.#explicitModeSet) {
                this.connectDirectly();
            } else {
                this.#log('Not falling back because mode was explicitly set');
                this.connectionState = STATE_ERROR;
                this.#triggerCallback(EVENT_ERROR, { message: 'SharedWorker initialization failed' });
            }
        }
    }

    /**
     * Connect directly to WebSocket server - Made public for external access
     */
    connectDirectly() {
        // Clean up any existing connections
        this.#cleanupCurrentConnection();
        
        // Direct mode doesn't use BroadcastChannel - ensure it's not initialized
        if (this.broadcastChannel) {
            console.log('[WS] WARNING: BroadcastChannel still exists after cleanup');
        }
        
        // Direct mode uses a tab-specific connection ID (not shared via localStorage)
        // This prevents conflicts when other tabs are using SharedWorker with the shared ID
        const tabSpecificId = this.#generateConnectionId();
        
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${protocol}//${window.location.host}${WS_CONFIG.wsRoute}?connid=${tabSpecificId}`;
        
        this.#socket = new WebSocket(url);
        this.connectionMode = MODE_DIRECT;
        
        this.#socket.onopen = () => {
            console.log('[WS] Direct WebSocket connected');
            this.connectionState = STATE_CONNECTED;
            this.#reconnectAttempts = 0;
            this.#triggerCallback(EVENT_OPEN);
        };
        
        this.#socket.onclose = () => {
            console.log('[WS] Direct WebSocket closed');
            this.connectionState = STATE_DISCONNECTED;
            this.#triggerCallback(EVENT_CLOSE);
            
            if (this.#options.reconnectOnDisconnect && !this.#explicitModeSet) {
                this.#attemptReconnect();
            }
        };
        
        this.#socket.onerror = (error) => {
            this.#log('Direct WebSocket error', error);
            this.#triggerCallback(EVENT_ERROR, error);
        };
        
        this.#socket.onmessage = (event) => {
            // Split on newlines in case server batches messages
            const messages = event.data.split('\n').filter(msg => msg.trim());
            messages.forEach(message => {
                this.#triggerCallback(EVENT_MESSAGE, message);
            });
        };
    }

    /**
     * Clean up the current connection before establishing a new one
     */
    #cleanupCurrentConnection() {
        // Set state to disconnected
        this.connectionState = STATE_DISCONNECTED;
        
        // Clean up worker
        if (this.#worker) {
            try {
                // Send disconnect message to the worker
                this.#worker.port.postMessage({ type: WORKER_DISCONNECT });
                // Close the port connection
                this.#worker.port.close();
                
                // Terminate the worker if possible (not all browsers support this)
                if (typeof this.#worker.terminate === 'function') {
                    this.#worker.terminate();
                }
                
                // Force worker to be garbage collected
                console.log('[WS] Worker connection terminated');
            } catch (e) {
                console.log('[WS] Error cleaning up worker', e);
            }
            this.#worker = null;
        }
        
        // Clean up direct connection
        if (this.#socket) {
            try{
                this.#socket.close();
            } catch (e) {
                console.log('[WS] Error cleaning up direct socket', e);
            }
            this.#socket = null;
        }
        
        // Kill any pending reconnect attempts
        if (this.#reconnectTimeout) {
            clearTimeout(this.#reconnectTimeout);
            this.#reconnectTimeout = null;
        }
        
        // Close BroadcastChannel when switching modes
        if (this.broadcastChannel) {
            try {
                if (this.#isPrimaryConnection) {
                    this.broadcastChannel.postMessage({
                        type: COORD_PRIMARY_DISCONNECTED,
                        id: this.#coordinationId,
                        timestamp: Date.now()
                    });
                }
                this.broadcastChannel.close();
                this.broadcastChannel = null;
                console.log('[WS] BroadcastChannel closed during cleanup');
            } catch (e) {
                console.log('[WS] Error closing BroadcastChannel', e);
            }
        }
    }

    /**
     * Remove message listener for iframe communication
     */
    #removeMessageListener() {
        if (this.#messageListener) {
            window.removeEventListener('message', this.#messageListener);
            this.#messageListener = null;
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
        if (this.broadcastChannel && !this.#isPrimaryConnection) {
            console.log('[WS] Sending message via coordination channel (secondary tab)');
            
            // Check if we already have a pending request for this message
            if (!this.#pendingSendRequests) {
                this.#pendingSendRequests = new Map();
            }
            
            const messageKey = message;
            const now = Date.now();
            
            // Check if we've recently sent this exact message
            if (this.#pendingSendRequests.has(messageKey)) {
                const lastRequest = this.#pendingSendRequests.get(messageKey);
                if (now - lastRequest.timestamp < 1000) { // Within 1 second
                    console.log('[WS] Duplicate send request detected, ignoring');
                    return true;
                }
            }
            
            // Generate a unique request ID to track responses
            const requestId = `${this.#coordinationId}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
            
            // Store this request to prevent duplicates
            this.#pendingSendRequests.set(messageKey, {
                requestId: requestId,
                timestamp: now
            });
            
            // Clean up old requests after 5 seconds
            setTimeout(() => {
                if (this.#pendingSendRequests) {
                    this.#pendingSendRequests.delete(messageKey);
                }
            }, 5000);
            
            // Send only once, with proper identification
            this.broadcastChannel.postMessage({
                type: COORD_SEND_REQUEST,
                requestId: requestId,
                senderId: this.#coordinationId,  // Add sender ID for better filtering
                id: this.#coordinationId,
                message: message,
                timestamp: now
            });
            return true; // Return early, don't try to send directly
        }
        
        // For primary connection or when coordination is not active
        if (this.connectionState !== STATE_CONNECTED) {
            this.#log('Cannot send message, not connected');
            return false;
        }
        
        switch (this.connectionMode) {
            case MODE_WORKER:
                this.#worker.port.postMessage({
                    type: WORKER_SEND,
                    data: message
                });
                break;
                
            case MODE_DIRECT:
                if (this.#socket && this.#socket.readyState === WebSocket.OPEN) {
                    this.#socket.send(message);
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
    #handleWorkerMessage(data) {
        switch (data.type) {
            case WORKER_CONNECTED:
                console.log('[WS] WebSocket connected via SharedWorker');
                this.connectionState = STATE_CONNECTED;
                this.#triggerCallback(EVENT_OPEN);
                break;
                
            case WORKER_DISCONNECTED:
                console.log('[WS] WebSocket disconnected via SharedWorker');
                this.connectionState = STATE_DISCONNECTED;
                this.#triggerCallback(EVENT_CLOSE);
                
                if (this.#options.reconnectOnDisconnect) {
                    this.#worker.port.postMessage({
                        type: WORKER_RECONNECT,
                        connectionId: this.connectionId
                    });
                }
                break;
                
            case WORKER_MESSAGE:
                // Process the message directly in primary tab
                this.#triggerCallback(EVENT_MESSAGE, data.data);
                
                // Only relay to other tabs if we're the primary
                if (this.broadcastChannel && this.#isPrimaryConnection) {
                    this.broadcastChannel.postMessage({
                        type: COORD_MESSAGE_RELAY,
                        id: this.#coordinationId,
                        message: data.data,
                        timestamp: Date.now()
                    });
                }
                break;
                
            case WORKER_ERROR:
                this.#log('WebSocket error via SharedWorker', data.error);
                this.#triggerCallback(EVENT_ERROR, data.error);
                break;
                
            default:
                this.#log('Unknown message from SharedWorker', data);
        }
    }

    /**
     * Handle coordination messages
     */
    #handleCoordinationMessage(data) {
        try {
            if (!data || !data.type) {
                console.log('[WS] Received invalid coordination message', data);
                return;
            }
            
            // Filter out our own messages for certain types
            if ([COORD_SEND_REQUEST, COORD_SEND_RESPONSE].includes(data.type) && data.senderId === this.#coordinationId) {
                // Ignore our own send requests and responses
                return;
            }
            
            switch (data.type) {
                case COORD_PRESENCE:
                case COORD_HEARTBEAT:
                    // Add/update this tab in known tabs list
                    if (this.#knownTabs && data.id !== this.#coordinationId) {
                        const existing = this.#knownTabs.has(data.id);
                        this.#knownTabs.set(data.id, {
                            id: data.id,
                            isPrimary: data.isPrimary,
                            lastSeen: Date.now()
                        });
                        
                        // Broadcast updated tabs list if this is a new tab
                        if (!existing) {
                            this.#broadcastTabsUpdate();
                        }
                    }
                    break;
                    
                case COORD_ELECTION:
                    // If our ID is "greater", we object and start a new election
                    if (this.#coordinationId > data.id) {
                        this.broadcastChannel.postMessage({
                            type: COORD_ELECTION_OBJECTION,
                            id: this.#coordinationId,
                            timestamp: Date.now()
                        });
                        
                        // Start our own election
                        setTimeout(() => this.#initiateElection(), 100);
                    }
                    break;
                    
                case COORD_ELECTION_OBJECTION:
                    // Someone objected to our election, cancel our timeout
                    if (this.#electionTimeout) {
                        clearTimeout(this.#electionTimeout);
                        this.#electionTimeout = null;
                    }
                    break;
                    
                case COORD_PRIMARY_ELECTED:
                    // Someone else became primary
                    if (data.id !== this.#coordinationId) {
                        this.#becomeSecondary();
                    }
                    break;
                    
                case COORD_PRIMARY_DISCONNECTED:
                    // Primary connection closed, initiate new election
                    if (data.id !== this.#coordinationId) {
                        setTimeout(() => this.#initiateElection(), 100 + Math.random() * 400);
                    }
                    break;
                    
                case COORD_MESSAGE_RELAY:
                    // Relay messages between tabs - only process if this is a secondary connection
                    if (data.id !== this.#coordinationId) {
                        // Check if we're a secondary tab before processing the relay
                        if (!this.#isPrimaryConnection) {
                            console.log('[WS] Received relayed message in secondary tab');
                            this.#triggerCallback(EVENT_MESSAGE, data.message);
                        } else {
                            console.log('[WS] Ignoring relayed message in primary tab (already received directly)');
                        }
                    }
                    break;
                    
                case COORD_SEND_REQUEST:
                    // Handle send requests from secondary tabs when we're primary
                    // Check both that we're primary AND this isn't our own message
                    if (this.#isPrimaryConnection && data.id !== this.#coordinationId && data.senderId !== this.#coordinationId) {
                        console.log('[WS] Received send request from secondary tab');
                        
                        // Use the request ID for deduplication
                        const messageId = data.requestId;
                        
                        if (!this.#recentlySentMessages) {
                            this.#recentlySentMessages = new Set();
                        }
                        
                        if (this.#recentlySentMessages.has(messageId)) {
                            console.log('[WS] Duplicate send request detected, ignoring');
                            // Still send response to avoid hanging the secondary tab
                            this.broadcastChannel.postMessage({
                                type: COORD_SEND_RESPONSE,
                                requestId: data.requestId,
                                targetId: data.senderId,  // Target specific sender
                                senderId: this.#coordinationId,
                                success: false,
                                duplicate: true,
                                timestamp: Date.now(),
                                id: this.#coordinationId
                            });
                            return;
                        }
                        
                        // Add to recent messages set
                        this.#recentlySentMessages.add(messageId);
                        
                        // Clean up old message IDs after 10 seconds
                        setTimeout(() => {
                            if (this.#recentlySentMessages) {
                                this.#recentlySentMessages.delete(messageId);
                            }
                        }, 10000);
                        
                        // Forward the message to the actual WebSocket
                        let success = false;
                        
                        // Only send if we have an active connection
                        if (this.connectionState !== STATE_CONNECTED) {
                            this.#log('Cannot forward message, primary not connected');
                            this.broadcastChannel.postMessage({
                                type: COORD_SEND_RESPONSE,
                                requestId: data.requestId,
                                targetId: data.senderId,
                                senderId: this.#coordinationId,
                                success: false,
                                error: 'not_connected',
                                timestamp: Date.now(),
                                id: this.#coordinationId
                            });
                            return;
                        }
                        
                        // Send directly to the actual socket
                        switch (this.connectionMode) {
                            case MODE_WORKER:
                                if (this.#worker) {
                                    this.#worker.port.postMessage({
                                        type: WORKER_SEND,
                                        data: data.message
                                    });
                                    success = true;
                                }
                                break;
                                
                            case MODE_DIRECT:
                                if (this.#socket && this.#socket.readyState === WebSocket.OPEN) {
                                    this.#socket.send(data.message);
                                    success = true;
                                }
                                break;
                        }
                        
                        // Send response targeted to the specific sender
                        this.broadcastChannel.postMessage({
                            type: COORD_SEND_RESPONSE,
                            requestId: data.requestId,
                            targetId: data.senderId,
                            senderId: this.#coordinationId,
                            success: success,
                            timestamp: Date.now(),
                            id: this.#coordinationId
                        });
                    }
                    break;
                
                case COORD_SEND_RESPONSE:
                    // Only process responses targeted to us
                    if (!this.#isPrimaryConnection && data.targetId === this.#coordinationId) {
                        if (data.duplicate) {
                            console.log('[WS] Send request was duplicate');
                        } else if (data.error === 'not_connected') {
                            console.log('[WS] Send request failed: primary not connected');
                        } else {
                            console.log(`[WS] Send request ${data.success ? 'succeeded' : 'failed'}`);
                        }
                    }
                    break;
                    
                case COORD_TABS_STATUS:
                    // Update with remote tab's view of all tabs
                    if (this.#knownTabs && data.id !== this.#coordinationId && data.tabs) {
                        let tabsChanged = false;
                        
                        data.tabs.forEach(tab => {
                            if (tab.id !== this.#coordinationId) { // Don't overwrite our own entry
                                const existing = this.#knownTabs.has(tab.id);
                                const currentTab = existing ? this.#knownTabs.get(tab.id) : null;
                                
                                if (!existing || (currentTab && currentTab.isPrimary !== tab.isPrimary)) {
                                    this.#knownTabs.set(tab.id, tab);
                                    tabsChanged = true;
                                }
                            }
                        });
                        
                        if (tabsChanged) {
                            this.#triggerCoordinationCallback(COORD_CB_TABS_UPDATED, Array.from(this.#knownTabs.values()));
                        }
                    }
                    break;
                    
                default:
                    this.#log(`Unknown coordination message type: ${data.type}`, data);
                    break;
            }
        } catch (error) {
            this.#log('Error handling coordination message', error);
        }
    }

    /**
     * Attempt to reconnect with exponential backoff
     */
    #attemptReconnect() {
        if (this.#reconnectAttempts >= this.#options.maxReconnectAttempts) {
            this.#log('Maximum reconnection attempts reached');
            return;
        }
        
        const delay = Math.min(1000 * Math.pow(2, this.#reconnectAttempts), 30000);
        this.#reconnectAttempts++;
        
        console.log(`[WS] Attempting to reconnect in ${delay}ms (attempt ${this.#reconnectAttempts}`);
        
        setTimeout(() => {
            if (this.connectionState === STATE_DISCONNECTED) {
                this.connectDirectly();
            }
        }, delay);
    }

    /**
     * Register event callbacks
     */
    on(event, callback) {
        if (this.#callbacks[event]) {
            this.#callbacks[event].push(callback);
        }
        return this;
    }

    /**
     * Trigger callbacks for an event
     * Simplified deduplication for message events
     */
    #triggerCallback(event, data) {
        if (event === EVENT_MESSAGE) {
            // Simplified deduplication - only check for exact same message within 1 second
            const messageKey = typeof data === 'string' ? data : JSON.stringify(data);
            
            if (!this.#recentMessages) {
                this.#recentMessages = new Map();
            }
            
            const now = Date.now();
            const lastSeen = this.#recentMessages.get(messageKey);
            
            // Only check for duplicates within 1 second window
            if (lastSeen && now - lastSeen < 1000) {
                return;
            }
            
            // Store the message timestamp
            this.#recentMessages.set(messageKey, now);
            
            // Clean up old messages every 10 seconds
            if (!this.#messageCleanupInterval) {
                this.#messageCleanupInterval = setInterval(() => {
                    const expiry = Date.now() - 5000; // Keep messages for 5 seconds
                    if (this.#recentMessages) {
                        this.#recentMessages.forEach((timestamp, msg) => {
                            if (timestamp < expiry) {
                                this.#recentMessages.delete(msg);
                            }
                        });
                    }
                }, 10000);
            }
        }
        
        // Existing callback triggering code
        if (this.#callbacks[event]) {
            this.#callbacks[event].forEach(callback => {
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
    #getStoredConnectionId() {
        let id = localStorage.getItem(this.#options.connIdStorageKey);
        if (!id) {
            id = this.#generateConnectionId();
            localStorage.setItem(this.#options.connIdStorageKey, id);
        }
        return id;
    }

    /**
     * Reset the connection ID to a new value
     * Useful when you want to force a new connection
     */
    #resetConnectionId() {
        this.connectionId = this.#generateConnectionId();
        localStorage.setItem(this.#options.connIdStorageKey, this.connectionId);
        return this.connectionId;
    }

    /**
     * Disconnect WebSocket
     */
    disconnect(suppressEvents = false) {
        this.connectionState = STATE_DISCONNECTED;
        
        switch (this.connectionMode) {
            case MODE_WORKER:
                if (this.#worker) {
                    try {
                        this.#worker.port.postMessage({ type: WORKER_DISCONNECT });
                        this.#worker.port.close();
                    } catch (e) {
                        console.log('[WS] Error closing worker', e);
                    }
                    this.#worker = null;
                }
                break;
                
            case MODE_DIRECT:
                if (this.#socket) {
                    try {
                        this.#socket.close();
                    } catch (e) {
                        console.log('[WS] Error closing socket', e);
                    }
                    this.#socket = null;
                }
                break;
        }
        
        // If this is the primary connection disconnecting and coordination is enabled,
        // notify other tabs unless suppressed (for coordination-managed disconnects)
        if (!suppressEvents && this.#isPrimaryConnection && this.broadcastChannel) {
            this.broadcastChannel.postMessage({
                type: COORD_PRIMARY_DISCONNECTED,
                id: this.#coordinationId,
                timestamp: Date.now()
            });
        }
        
        this.#triggerCallback(EVENT_CLOSE);
    }

    /**
     * Completely reset the connection
     */
    resetConnection() {
        this.disconnect();
        this.clearPreferredMode();
        this.connectionId = this.#resetConnectionId();
        return this.connect();
    }

    /**
     * Log message if debug is enabled
     */
    #log(message, data) {
        if (this.#options.debug) {
            if (data) {
                window.console.log(`[WebSocketManager] ${message}`, data);
            } else {
                window.console.log(`[WebSocketManager] ${message}`);
            }
        }
    }

    /**
     * Try to clear any existing SharedWorkers
     * Note: This is a best-effort approach as browsers limit what we can do with workers
     */
    #clearSharedWorkers() {
        console.log('[WS] Attempting to clear SharedWorkers');
        
        // Create a temporary worker to send a global shutdown message
        try {
            const tempWorker = new SharedWorker(WS_CONFIG.workerRoute);
            tempWorker.port.start();
            tempWorker.port.postMessage({ type: WORKER_GLOBAL_SHUTDOWN });
            
            // Close the port after sending the message
            setTimeout(() => {
                try {
                    tempWorker.port.close();
                } catch (e) {
                    // Ignore errors on cleanup
                }
            }, 100);
        } catch (e) {
            console.log('[WS] Error creating temporary worker for cleanup', e);
        }
        
        // Additional fallback - notify the server about mode change
        // to help it clean up orphaned connections
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const url = `${protocol}//${window.location.host}${WS_CONFIG.wsRoute}?connid=${this.connectionId}&cleanup=1`;
            
            const cleanupSocket = new WebSocket(url);
            cleanupSocket.onopen = () => {
                cleanupSocket.send(JSON.stringify({
                    type: MSG_TYPE_MODE_CHANGE,
                    previousMode: MODE_WORKER,
                    newMode: this.connectionMode
                }));
                
                // Close immediately after sending
                setTimeout(() => cleanupSocket.close(), 50);
            };
        } catch (e) {
            console.log('[WS] Error sending cleanup notification', e);
        }
    }

    /**
     * Initialize connection coordination using BroadcastChannel
     */
    initConnectionCoordination() {
        // Check if BroadcastChannel is supported
        if (typeof BroadcastChannel === 'undefined') {
            console.log('[WS] BroadcastChannel not supported, coordination disabled');
            return false;
        }
        
        try {
            // Create or get the broadcast channel
            this.broadcastChannel = new BroadcastChannel(this.#options.coordinationChannel);
            
            // Use arrow function to preserve 'this' context
            this.broadcastChannel.onmessage = (event) => {
                // Add global message deduplication
                if (!this.#recentCoordinationMessages) {
                    this.#recentCoordinationMessages = new Map();
                }
                
                const messageKey = JSON.stringify(event.data);
                const now = Date.now();
                
                // Check if we've seen this exact message recently
                const lastSeen = this.#recentCoordinationMessages.get(messageKey);
                if (lastSeen && now - lastSeen < 100) { // Within 100ms
                    // Ignore duplicate coordination message
                    return;
                }
                
                // Store this message
                this.#recentCoordinationMessages.set(messageKey, now);
                
                // Clean up old messages periodically
                if (!this.#coordinationCleanupInterval) {
                    this.#coordinationCleanupInterval = setInterval(() => {
                        const expiry = Date.now() - 1000; // Keep for 1 second
                        this.#recentCoordinationMessages.forEach((timestamp, msg) => {
                            if (timestamp < expiry) {
                                this.#recentCoordinationMessages.delete(msg);
                            }
                        });
                    }, 5000);
                }
                
                this.#handleCoordinationMessage(event.data);
            };
            
            // Start the coordination process
            this.#startCoordination();
            
            // Trigger coordination enabled event
            this.#triggerCoordinationCallback(COORD_CB_ENABLED);
            
            return true;
        } catch (e) {
            this.#log('Error initializing coordination', e);
            return false;
        }
    }
    
    /**
     * Start the coordination process
     */
    #startCoordination() {
        // Clear any existing intervals/timeouts
        this.#clearCoordinationTimers();
        
        // Send presence announcement
        this.#broadcastPresence();
        
        // Start heartbeat interval
        this.#heartbeatInterval = setInterval(() => {
            this.#broadcastHeartbeat();
        }, this.#options.coordinationHeartbeat);
        
        // Initiate election
        this.#initiateElection();
        
        // Track all known tabs
        this.#knownTabs = new Map();
        this.#knownTabs.set(this.#coordinationId, {
            id: this.#coordinationId,
            isPrimary: this.#isPrimaryConnection,
            lastSeen: Date.now()
        });
        
        // Add periodic tab cleanup and status update
        setInterval(() => {
            this.#cleanupStaleTabsAndUpdateStatus();
        }, this.#options.coordinationHeartbeat * 2);
        
        // Handle page visibility changes
        document.addEventListener('visibilitychange', () => {
            if (document.visibilityState === 'visible') {
                console.log('[WS] Tab became visible, re-synchronizing coordination state');
                
                // If we were primary but tab was backgrounded, we might have been throttled
                // Assume we're no longer primary and let election determine the new state
                if (this.#isPrimaryConnection) {
                    console.log('[WS] Was primary but may have been throttled, becoming secondary and re-electing');
                    this.#isPrimaryConnection = false;
                }
                
                // Re-announce presence and participate in election
                this.#broadcastPresence();
                this.#initiateElection();
            }
        });
        
        // Handle before unload to notify other tabs
        window.addEventListener('beforeunload', () => {
            if (this.#isPrimaryConnection) {
                this.broadcastChannel.postMessage({
                    type: COORD_PRIMARY_DISCONNECTED,
                    id: this.#coordinationId,
                    timestamp: Date.now()
                });
            }
        });
    }
    
    /**
     * Clear coordination timers
     */
    #clearCoordinationTimers() {
        if (this.#heartbeatInterval) {
            clearInterval(this.#heartbeatInterval);
            this.#heartbeatInterval = null;
        }
        
        if (this.#electionTimeout) {
            clearTimeout(this.#electionTimeout);
            this.#electionTimeout = null;
        }
    }
    
    /**
     * Broadcast presence to other tabs
     */
    #broadcastPresence() {
        if (!this.broadcastChannel) return;
        
        this.broadcastChannel.postMessage({
            type: COORD_PRESENCE,
            id: this.#coordinationId,
            timestamp: Date.now(),
            isPrimary: this.#isPrimaryConnection,
            connectionState: this.connectionState
        });
        
        // Update this tab in known tabs
        if (this.#knownTabs) {
            this.#knownTabs.set(this.#coordinationId, {
                id: this.#coordinationId,
                isPrimary: this.#isPrimaryConnection,
                lastSeen: Date.now()
            });
            
            // Notify about tabs update
            this.#broadcastTabsUpdate();
        }
    }
    
    /**
     * Clean up stale tabs and update status
     */
    #cleanupStaleTabsAndUpdateStatus() {
        if (!this.#knownTabs) return;
        
        const now = Date.now();
        const cutoff = now - (this.#options.assumeDisconnectedAfter * 2);
        
        // Remove stale tabs
        let tabsChanged = false;
        this.#knownTabs.forEach((tab, id) => {
            if (tab.lastSeen < cutoff) {
                this.#knownTabs.delete(id);
                tabsChanged = true;
            }
        });
        
        if (tabsChanged) {
            this.#broadcastTabsUpdate();
        }
    }
    
    /**
     * Broadcast tabs update
     */
    #broadcastTabsUpdate() {
        if (!this.#knownTabs) return;
        
        const tabsList = Array.from(this.#knownTabs.values());
        
        // Notify local listeners
        this.#triggerCoordinationCallback(COORD_CB_TABS_UPDATED, tabsList);
        
        // Broadcast tabs status to other tabs
        if (this.broadcastChannel) {
            this.broadcastChannel.postMessage({
                type: COORD_TABS_STATUS,
                id: this.#coordinationId,
                timestamp: Date.now(),
                tabs: tabsList
            });
        }
    }
    
    /**
     * Broadcast heartbeat to other tabs
     */
    #broadcastHeartbeat() {
        if (!this.broadcastChannel) return;
        
        this.broadcastChannel.postMessage({
            type: COORD_HEARTBEAT,
            id: this.#coordinationId,
            timestamp: Date.now(),
            isPrimary: this.#isPrimaryConnection,
            connectionState: this.connectionState
        });
    }
    
    /**
     * Initiate election to determine primary connection
     */
    #initiateElection() {
        if (!this.broadcastChannel) return;
        
        // Send election message
        this.broadcastChannel.postMessage({
            type: COORD_ELECTION,
            id: this.#coordinationId,
            timestamp: Date.now()
        });
        
        // Set timeout to declare self as primary if no one objects
        this.#electionTimeout = setTimeout(() => {
            this.#becomePrimary();
        }, 500);
    }
    
    /**
     * Become the primary connection
     */
    #becomePrimary() {
        if (this.#isPrimaryConnection) return;
        
        console.log('[WS] Becoming primary connection');
        this.#isPrimaryConnection = true;
        
        // Announce primary status
        this.broadcastChannel.postMessage({
            type: COORD_PRIMARY_ELECTED,
            id: this.#coordinationId,
            timestamp: Date.now()
        });
        
        // Ensure connection is active
        if (this.connectionState !== STATE_CONNECTED) {
            this.connect();
        }
        
        // Update our entry in known tabs
        if (this.#knownTabs) {
            this.#knownTabs.set(this.#coordinationId, {
                id: this.#coordinationId,
                isPrimary: true,
                lastSeen: Date.now()
            });
            this.#broadcastTabsUpdate();
        }
        
        // Trigger became-primary event
        this.#triggerCoordinationCallback(COORD_CB_BECAME_PRIMARY);
    }
    
    /**
     * Become a secondary connection
     */
    #becomeSecondary() {
        if (!this.#isPrimaryConnection) return;
        
        console.log('[WS] Becoming secondary connection');
        this.#isPrimaryConnection = false;
        
        // Disconnect if connected (primary will handle the connection)
        if (this.connectionState === STATE_CONNECTED) {
            this.disconnect(true);
        }
        
        // Update our entry in known tabs
        if (this.#knownTabs) {
            this.#knownTabs.set(this.#coordinationId, {
                id: this.#coordinationId,
                isPrimary: false,
                lastSeen: Date.now()
            });
            this.#broadcastTabsUpdate();
        }
        
        // Trigger became-secondary event
        this.#triggerCoordinationCallback(COORD_CB_BECAME_SECONDARY);
    }
    
    /**
     * Register coordination event callbacks
     */
    onCoordinationEvent(event, callback) {
        if (!this.#coordinationCallbacks[event]) {
            this.#coordinationCallbacks[event] = [];
        }
        this.#coordinationCallbacks[event].push(callback);
        return this;
    }
    
    /**
     * Trigger coordination callbacks for an event
     */
    #triggerCoordinationCallback(event, data) {
        if (this.#coordinationCallbacks[event]) {
            this.#coordinationCallbacks[event].forEach(callback => {
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
    #generateInstanceId() {
        return Date.now().toString() + Math.random().toString(36).substring(2, 9);
    }
    
    /**
     * Generate a connection ID
     */
    #generateConnectionId() {
        return Date.now().toString() + Math.random().toString(36).substring(2, 9);
    }
    
    /**
     * Clean up and dispose all resources
     */
    dispose() {
        this.disconnect();
        
        // Clean up intervals
        if (this.#messageCleanupInterval) {
            clearInterval(this.#messageCleanupInterval);
            this.#messageCleanupInterval = null;
        }
        
        if (this.#coordinationCleanupInterval) {
            clearInterval(this.#coordinationCleanupInterval);
            this.#coordinationCleanupInterval = null;
        }
        
        // Clear message caches
        if (this.#recentMessages) {
            this.#recentMessages.clear();
        }
        
        if (this.#recentlySentMessages) {
            this.#recentlySentMessages.clear();
        }
        
        if (this.#recentCoordinationMessages) {
            this.#recentCoordinationMessages.clear();
        }
        
        // Clear pending send requests
        if (this.#pendingSendRequests) {
            this.#pendingSendRequests.clear();
        }
        
        if (this.broadcastChannel) {
            this.#clearCoordinationTimers();
            
            if (this.#isPrimaryConnection) {
                this.broadcastChannel.postMessage({
                    type: COORD_PRIMARY_DISCONNECTED,
                    id: this.#coordinationId,
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
        return this.#knownTabs ? Array.from(this.#knownTabs.values()) : [];
    }
    
    /**
     * Check if this tab is the primary connection
     */
    isPrimary() {
        return this.#isPrimaryConnection;
    }
}

/**
 * Create a WebSocketManager instance
 * @param {Object} options - Configuration options
 * @returns {Promise<WebSocketManager>} WebSocketManager instance (resolves when DOM is ready)
 */
export async function createWebSocketManager(options = {}) {
    return new Promise((resolve) => {
        // Don't create manager or resolve until DOM is ready
        const initManager = () => {
            const manager = new WebSocketManager(options);
            if (manager.initConnectionCoordination) {
                manager.initConnectionCoordination();
            }
            resolve(manager);
        };
        
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', initManager);
        } else {
            initManager();
        }
    });
}

// Export class and constants
export { 
    WebSocketManager,
    WS_CONFIG,
    STATE_DISCONNECTED,
    STATE_CONNECTING,
    STATE_CONNECTED,
    STATE_ERROR,
    MODE_WORKER,
    MODE_DIRECT,
    EVENT_MESSAGE,
    EVENT_OPEN,
    EVENT_ERROR,
    EVENT_CLOSE,
    COORD_CB_ENABLED,
    COORD_CB_BECAME_PRIMARY,
    COORD_CB_BECAME_SECONDARY,
    COORD_CB_TABS_UPDATED
};
// {{ end }}
