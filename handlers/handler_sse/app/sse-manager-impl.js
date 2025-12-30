/**
 * SSEManager - Coordinated Server-Sent Events Manager
 * 
 * Manages SSE connections across multiple tabs using BroadcastChannel.
 * Only one tab maintains an active SSE connection (primary), while other tabs
 * (secondary) receive events via BroadcastChannel coordination.
 * 
 * Features:
 * - Single active SSE connection across all tabs
 * - Automatic leader election
 * - Failover when primary tab closes
 * - Heartbeat monitoring
 */



const I_AM_PRIMARY = 100
const REQUEST_LEADER = 101
const LEADER_RESPONSE = 102
const HEARTBEAT = 103
const MESSAGE = 104
const DISCONNECTING = 105
const ELECTION = 106
const ELECTION_RESPONSE = 107

const STATE_DISCONNECTED = 0
const STATE_CONNECTING = 1
const STATE_CONNECTED = 2

const EVENT_MESSAGE = 0
const EVENT_OPEN = 1
const EVENT_ERROR = 2
const EVENT_CLOSE = 3
const EVENT_PRIMARY = 4
const EVENT_SECONDARY = 5

class SSEManager {
    #isPrimary = false;
    #connectionState = STATE_DISCONNECTED;
    #url;
    #eventSource = null;
    #broadcastChannel = null;
    #channelName;
    #heartbeatTimer = null;
    #electionTimer = null;
    #reconnectTimer = null;
    #failoverTimer = null;
    #lastHeartbeat = null;
    #callbacks_message;
    #callbacks_open;
    #callbacks_error;
    #callbacks_close;
    #callbacks_primary;
    #callbacks_secondary;
    #instanceId;
    #boundCleanup;
    
    constructor(url, options = {}) {
        this.#url = url;
        this.#instanceId = this.#generateInstanceId();
        this.#channelName = 'sse-coord-' + btoa(url).replace(/=/g, '');
        
        this.ops = {
            reconnect: options.reconnect ?? true,
            reconnectInterval: options.reconnectInterval ?? 5000,
            heartbeatInterval: options.heartbeatInterval ?? 2000,
            electionTimeout: options.electionTimeout ?? 500,
            failoverCheckInterval: options.failoverCheckInterval ?? 10000
        }
        
        // Callbacks
        this.#callbacks_message = [];
        this.#callbacks_open = [];
        this.#callbacks_error = [];
        this.#callbacks_close = [];
        this.#callbacks_primary = [];
        this.#callbacks_secondary = [];
        
        // Auto-cleanup on page unload
        this.#boundCleanup = () => this.disconnect();
        window.addEventListener('beforeunload', this.#boundCleanup);
        window.addEventListener('pagehide', this.#boundCleanup);
    }
    
    /**
     * Generate a unique instance ID
     */
    #generateInstanceId() {
        return Date.now().toString() + '-' + Math.random().toString(36).substr(2, 9);
    }
    
    /**
     * Connect to SSE with coordination
     */
    connect() {
        if (this.#connectionState !== STATE_DISCONNECTED) {
            console.warn('[SSEManager] Already connected or connecting');
            return;
        }
        
        // Ensure clean state
        if (this.#eventSource) {
            this.#eventSource.close();
            this.#eventSource = null;
        }
        
        this.#connectionState = STATE_CONNECTING;
        this.#initCoordination();
    }
    
    /**
     * Disconnect from SSE
     */
    disconnect() {
        console.log('[SSEManager] Disconnect called, closing connection');
        
        // Immediately disable reconnect
        this.ops.reconnect = false;
        
        // Clear all timers FIRST to prevent any reconnection attempts
        this.#clearTimers();
        
        // Force close SSE connection
        if (this.#eventSource) {
            console.log('[SSEManager] Closing EventSource, readyState:', this.#eventSource.readyState);
            
            // Remove all event handlers BEFORE closing to prevent error/close events from firing
            this.#eventSource.onopen = null;
            this.#eventSource.onmessage = null;
            this.#eventSource.onerror = null;
            
            // Close the connection
            this.#eventSource.close();
            console.log('[SSEManager] EventSource closed, new readyState:', this.#eventSource.readyState);
            this.#eventSource = null;
        }
        
        // Update state AFTER closing connection
        this.#connectionState = STATE_DISCONNECTED;
        this.#isPrimary = false;
        
        // Notify other tabs before closing channel
        if (this.#broadcastChannel) {
            this.#broadcastChannel.postMessage({
                type: DISCONNECTING,
                instanceId: this.#instanceId
            });
            this.#broadcastChannel.close();
            this.#broadcastChannel = null;
        }
        
        // Remove page unload listeners
        if (this.#boundCleanup) {
            window.removeEventListener('beforeunload', this.#boundCleanup);
            window.removeEventListener('pagehide', this.#boundCleanup);
        }
        
        this.#triggerCallback(EVENT_CLOSE);
        console.log('[SSEManager] Disconnect complete');
    }
    
    /**
     * Initialize BroadcastChannel coordination
     */
    #initCoordination() {
        try {
            this.#broadcastChannel = new BroadcastChannel(this.#channelName);
            
            // Handle messages from other tabs
            this.#broadcastChannel.onmessage = (event) => {
                this.#handleCoordinationMessage(event.data);
            };
            
            // Request current leader
            this.#broadcastChannel.postMessage({
                type: REQUEST_LEADER,
                instanceId: this.#instanceId
            });
            
            // Start election after delay
            this.#electionTimer = setTimeout(() => {
                this.#initiateElection();
            }, this.ops.electionTimeout);
            
        } catch (error) {
            console.warn('[SSEManager] BroadcastChannel not supported, connecting directly');
            this.#becomePrimary();
        }
    }
    
    /**
     * Handle coordination messages from BroadcastChannel
     */
    #handleCoordinationMessage(data) {
        switch (data.type) {
            case REQUEST_LEADER:
                if (this.#isPrimary) {
                    this.#broadcastChannel.postMessage({
                        type: LEADER_RESPONSE,
                        instanceId: this.#instanceId
                    });
                }
                break;
                
            case LEADER_RESPONSE:
                if (!this.#isPrimary && data.instanceId !== this.#instanceId) {
                    clearTimeout(this.#electionTimer);
                    this.#becomeSecondary();
                }
                break;
                
            case HEARTBEAT:
                if (!this.#isPrimary && data.instanceId !== this.#instanceId) {
                    this.#lastHeartbeat = Date.now();
                }
                break;
                
            case MESSAGE:
                if (!this.#isPrimary) {
                    this.#triggerCallback(EVENT_MESSAGE, data.message);
                }
                break;
                
            case DISCONNECTING:
                if (!this.#isPrimary && data.instanceId !== this.#instanceId) {
                    // Primary is closing, initiate election after a delay
                    clearTimeout(this.#electionTimer);
                    this.#electionTimer = setTimeout(() => this.#initiateElection(), 100);
                }
                break;
                
            case ELECTION:
                // Another tab is running for election
                if (data.instanceId !== this.#instanceId) {
                    // If the other tab has a lower ID (higher priority), it should win
                    if (data.instanceId < this.#instanceId) {
                        // We defer to the lower ID
                        if (this.#isPrimary) {
                            this.#downgradeToPrimary();
                        } else {
                            // Cancel our own election attempt
                            clearTimeout(this.#electionTimer);
                        }
                    } else {
                        // Our ID is lower (higher priority), we should win
                        // Respond to assert our claim
                        this.#broadcastChannel.postMessage({
                            type: ELECTION_RESPONSE,
                            instanceId: this.#instanceId
                        });
                    }
                }
                break;
                
            case ELECTION_RESPONSE:
                // Another tab is asserting its claim with lower ID
                if (data.instanceId < this.#instanceId && data.instanceId !== this.#instanceId) {
                    // Abort our election, they have priority
                    clearTimeout(this.#electionTimer);
                    if (this.#isPrimary) {
                        this.#downgradeToPrimary();
                    }
                }
                break;
                
            case I_AM_PRIMARY:
                // Another tab has claimed primary status
                if (data.instanceId !== this.#instanceId) {
                    if (this.#isPrimary) {
                        // Two primaries! Use instanceId to resolve
                        if (data.instanceId < this.#instanceId) {
                            // They win, we downgrade
                            this.#downgradeToPrimary();
                        }
                    } else {
                        // We're secondary, update last heartbeat
                        this.#lastHeartbeat = Date.now();
                    }
                }
                break;
        }
    }
    
    /**
     * Initiate election to become primary
     */
    #initiateElection() {
        if (!this.#broadcastChannel) {
            this.#becomePrimary();
            return;
        }
        
        // Broadcast our candidacy
        this.#broadcastChannel.postMessage({
            type: ELECTION,
            instanceId: this.#instanceId
        });
        
        // Wait to see if anyone with lower ID objects
        clearTimeout(this.#electionTimer);
        this.#electionTimer = setTimeout(() => {
            if (!this.#isPrimary) {
                this.#becomePrimary();
            }
        }, 500);
    }
    
    /**
     * Become the primary connection
     */
    #becomePrimary() {
        if (this.#isPrimary) return;
        
        this.#isPrimary = true;
        this.#connectionState = STATE_CONNECTED;
        
        // Announce that we are now primary
        if (this.#broadcastChannel) {
            this.#broadcastChannel.postMessage({
                type: I_AM_PRIMARY,
                instanceId: this.#instanceId
            });
        }
        
        this.#connectDirectly();
        
        // Start heartbeat
        this.#heartbeatTimer = setInterval(() => {
            if (this.#broadcastChannel && this.#isPrimary) {
                this.#broadcastChannel.postMessage({
                    type: HEARTBEAT,
                    instanceId: this.#instanceId
                });
            }
        }, this.ops.heartbeatInterval);
        
        this.#triggerCallback(EVENT_PRIMARY);
    }
    
    /**
     * Become a secondary connection
     */
    #becomeSecondary() {
        if (!this.#isPrimary && this.#eventSource) return;
        
        this.#isPrimary = false;
        this.#connectionState = STATE_CONNECTED;
        this.#lastHeartbeat = Date.now();
        
        // Monitor for primary failure
        this.#failoverTimer = setInterval(() => {
            const timeSinceLastHeartbeat = Date.now() - (this.#lastHeartbeat || 0);
            if (timeSinceLastHeartbeat > this.ops.failoverCheckInterval) {
                // Primary might be dead, initiate election
                this.#initiateElection();
            }
        }, this.ops.failoverCheckInterval);
        
        this.#triggerCallback(EVENT_SECONDARY);
        this.#triggerCallback(EVENT_OPEN);
    }
    
    /**
     * Downgrade from primary to secondary
     */
    #downgradeToPrimary() {
        if (this.#eventSource) {
            this.#eventSource.close();
            this.#eventSource = null;
        }
        
        if (this.#heartbeatTimer) {
            clearInterval(this.#heartbeatTimer);
            this.#heartbeatTimer = null;
        }
        
        this.#isPrimary = false;
        this.#becomeSecondary();
    }
    
    /**
     * Connect directly to SSE endpoint
     */
    #connectDirectly() {
        // Close any existing connection first
        if (this.#eventSource) {
            this.#eventSource.onopen = null;
            this.#eventSource.onmessage = null;
            this.#eventSource.onerror = null;
            this.#eventSource.close();
            this.#eventSource = null;
        }
        
        this.#eventSource = new EventSource(this.#url);
        
        this.#eventSource.onopen = () => {
            this.#connectionState = STATE_CONNECTED;
            this.#triggerCallback(EVENT_OPEN);
        };
        
        this.#eventSource.onmessage = (event) => {
            // Trigger local callback
            this.#triggerCallback(EVENT_MESSAGE, event.data);
            
            // Broadcast to other tabs if primary
            if (this.#isPrimary && this.#broadcastChannel) {
                this.#broadcastChannel.postMessage({
                    type: MESSAGE,
                    message: event.data,
                    instanceId: this.#instanceId
                });
            }
        };
        
        this.#eventSource.onerror = (error) => {
            this.#triggerCallback(EVENT_ERROR, error);
            
            if (this.#eventSource.readyState === EventSource.CLOSED) {
                this.#handleConnectionLost();
            }
        };
    }
    
    /**
     * Handle connection loss
     */
    #handleConnectionLost() {
        this.#connectionState = STATE_DISCONNECTED;
        
        if (this.#eventSource) {
            this.#eventSource.close();
            this.#eventSource = null;
        }
        
        this.#triggerCallback(EVENT_CLOSE);
        
        // Attempt reconnect if enabled
        if (this.ops.reconnect && this.#isPrimary) {
            this.#reconnectTimer = setTimeout(() => {
                if (this.#isPrimary) {
                    this.#connectDirectly();
                }
            }, this.ops.reconnectInterval);
        }
    }
    
    /**
     * Clear all timers
     */
    #clearTimers() {
        if (this.#heartbeatTimer) {
            clearInterval(this.#heartbeatTimer);
            this.#heartbeatTimer = null;
        }
        
        if (this.#electionTimer) {
            clearTimeout(this.#electionTimer);
            this.#electionTimer = null;
        }
        
        if (this.#reconnectTimer) {
            clearTimeout(this.#reconnectTimer);
            this.#reconnectTimer = null;
        }
        
        if (this.#failoverTimer) {
            clearInterval(this.#failoverTimer);
            this.#failoverTimer = null;
        }
    }
    
    /**
     * Register event callback
     */
    on(event, callback) {
        switch(event) {
            case EVENT_MESSAGE: this.#callbacks_message.push(callback); break;
            case EVENT_OPEN: this.#callbacks_open.push(callback); break;
            case EVENT_ERROR: this.#callbacks_error.push(callback); break;
            case EVENT_CLOSE: this.#callbacks_close.push(callback); break;
            case EVENT_PRIMARY: this.#callbacks_primary.push(callback); break;
            case EVENT_SECONDARY: this.#callbacks_secondary.push(callback); break;
        }
    }
    
    /**
     * Trigger event callbacks
     */
    #triggerCallback(event, data) {
        let callbacks;
        switch(event) {
            case EVENT_MESSAGE: callbacks = this.#callbacks_message; break;
            case EVENT_OPEN: callbacks = this.#callbacks_open; break;
            case EVENT_ERROR: callbacks = this.#callbacks_error; break;
            case EVENT_CLOSE: callbacks = this.#callbacks_close; break;
            case EVENT_PRIMARY: callbacks = this.#callbacks_primary; break;
            case EVENT_SECONDARY: callbacks = this.#callbacks_secondary; break;
        }
        if (callbacks) {
            callbacks.forEach(callback => callback(data));
        }
    }
    
    /**
     * Check if this tab is the primary connection
     */
    isPrimaryConnection() {
        return this.#isPrimary;
    }
    
    /**
     * Get current connection state
     */
    getConnectionState() {
        return this.#connectionState;
    }
    
    /**
     * Get current state
     */
    getState() {
        const state = {};
        state["connectionState"] = this.#connectionState;
        state["isPrimary"] = this.#isPrimary;
        state["instanceId"] = this.#instanceId;
        state["url"] = this.#url;
        return state;
    }
}


// Export for ES module usage
export { SSEManager };