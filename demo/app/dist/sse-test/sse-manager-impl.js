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



class SSEManager {
    constructor(url) {
        this.url = url;
        
        // SSE connection
        this.eventSource = null;
        this.connectionState = 'disconnected';
        
        // Coordination
        this.broadcastChannel = null;
        this.isPrimary = false;
        this.instanceId = this.generateInstanceId();
        this.channelName = 'sse-coord-' + btoa(url).replace(/=/g, '');
        
        // Timers
        this.heartbeatTimer = null;
        this.electionTimer = null;
        this.reconnectTimer = null;
        this.failoverTimer = null;
        this.lastHeartbeat = null;
        
        // Options with defaults
        this.reconnect = true;
        this.reconnectInterval = 5000;
        this.heartbeatInterval = 2000;
        this.electionTimeout = 500;
        this.failoverCheckInterval = 10000;
        
        // Callbacks
        this.callbacks = {
            message: [],
            open: [],
            error: [],
            close: [],
            primary: [],
            secondary: []
        };
        
        // Public API - accessed by index via $ property to avoid minification issues
        // 0: connect, 1: disconnect, 2: on, 3: isPrimaryConnection, 4: getConnectionState, 5: getState
        // 6: gs_reconnect, 7: gs_reconnectInterval, 8: gs_heartbeatInterval, 9: gs_electionTimeout, 10: gs_failoverCheckInterval
        this.$ = [
            this.connect.bind(this),
            this.disconnect.bind(this),
            this.on.bind(this),
            this.isPrimaryConnection.bind(this),
            this.getConnectionState.bind(this),
            this.getState.bind(this),
            this.gs_reconnect.bind(this),
            this.gs_reconnectInterval.bind(this),
            this.gs_heartbeatInterval.bind(this),
            this.gs_electionTimeout.bind(this),
            this.gs_failoverCheckInterval.bind(this)
        ];
    }
    
    // Combined getter/setter for options (minification-safe)
    gs_reconnect(val) { this.reconnect = val === undefined ? this.reconnect : val; return this.reconnect; }
    gs_reconnectInterval(val) { this.reconnectInterval = val === undefined ? this.reconnectInterval : val; return this.reconnectInterval; }
    gs_heartbeatInterval(val) { this.heartbeatInterval = val === undefined ? this.heartbeatInterval : val; return this.heartbeatInterval; }
    gs_electionTimeout(val) { this.electionTimeout = val === undefined ? this.electionTimeout : val; return this.electionTimeout; }
    gs_failoverCheckInterval(val) { this.failoverCheckInterval = val === undefined ? this.failoverCheckInterval : val; return this.failoverCheckInterval; }
    
 
    /**
     * Generate a unique instance ID
     */
    generateInstanceId() {
        return Date.now().toString() + '-' + Math.random().toString(36).substr(2, 9);
    }
    
    /**
     * Connect to SSE with coordination
     */
    connect() {
        if (this.connectionState !== 'disconnected') {
            console.warn('[SSEManager] Already connected or connecting');
            return;
        }
        
        // Ensure clean state
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
        
        this.connectionState = 'connecting';
        this.initCoordination();
    }
    
    /**
     * Disconnect from SSE
     */
    disconnect() {
        this.gs_reconnect(false);
        this.connectionState = 'disconnected';
        
        // Clear all timers
        this.clearTimers();
        
        // Close SSE connection properly
        if (this.eventSource) {
            this.eventSource.onopen = null;
            this.eventSource.onmessage = null;
            this.eventSource.onerror = null;
            this.eventSource.close();
            this.eventSource = null;
        }
        
        // Notify other tabs before closing channel
        if (this.broadcastChannel) {
            this.broadcastChannel.postMessage({
                type: 'DISCONNECTING',
                instanceId: this.instanceId
            });
            this.broadcastChannel.close();
            this.broadcastChannel = null;
        }
        
        this.isPrimary = false;
        this.triggerCallback('close');
    }
    
    /**
     * Initialize BroadcastChannel coordination
     */
    initCoordination() {
        try {
            this.broadcastChannel = new BroadcastChannel(this.channelName);
            
            // Handle messages from other tabs
            this.broadcastChannel.onmessage = (event) => {
                this.handleCoordinationMessage(event.data);
            };
            
            // Request current leader
            this.broadcastChannel.postMessage({
                type: 'REQUEST_LEADER',
                instanceId: this.instanceId
            });
            
            // Start election after delay
            this.electionTimer = setTimeout(() => {
                this.initiateElection();
            }, this.gs_electionTimeout());
            
        } catch (error) {
            console.warn('[SSEManager] BroadcastChannel not supported, connecting directly');
            this.becomePrimary();
        }
    }
    
    /**
     * Handle coordination messages from BroadcastChannel
     */
    handleCoordinationMessage(data) {
        switch (data.type) {
            case 'REQUEST_LEADER':
                if (this.isPrimary) {
                    this.broadcastChannel.postMessage({
                        type: 'LEADER_RESPONSE',
                        instanceId: this.instanceId
                    });
                }
                break;
                
            case 'LEADER_RESPONSE':
                if (!this.isPrimary && data.instanceId !== this.instanceId) {
                    clearTimeout(this.electionTimer);
                    this.becomeSecondary();
                }
                break;
                
            case 'HEARTBEAT':
                if (!this.isPrimary && data.instanceId !== this.instanceId) {
                    this.lastHeartbeat = Date.now();
                }
                break;
                
            case 'MESSAGE':
                if (!this.isPrimary) {
                    this.triggerCallback('message', data.message);
                }
                break;
                
            case 'DISCONNECTING':
                if (!this.isPrimary && data.instanceId !== this.instanceId) {
                    // Primary is closing, initiate election after a delay
                    clearTimeout(this.electionTimer);
                    this.electionTimer = setTimeout(() => this.initiateElection(), 100);
                }
                break;
                
            case 'ELECTION':
                // Another tab is running for election
                if (data.instanceId !== this.instanceId) {
                    // If the other tab has a lower ID (higher priority), it should win
                    if (data.instanceId < this.instanceId) {
                        // We defer to the lower ID
                        if (this.isPrimary) {
                            this.downgradeToPrimary();
                        } else {
                            // Cancel our own election attempt
                            clearTimeout(this.electionTimer);
                        }
                    } else {
                        // Our ID is lower (higher priority), we should win
                        // Respond to assert our claim
                        this.broadcastChannel.postMessage({
                            type: 'ELECTION_RESPONSE',
                            instanceId: this.instanceId
                        });
                    }
                }
                break;
                
            case 'ELECTION_RESPONSE':
                // Another tab is asserting its claim with lower ID
                if (data.instanceId < this.instanceId && data.instanceId !== this.instanceId) {
                    // Abort our election, they have priority
                    clearTimeout(this.electionTimer);
                    if (this.isPrimary) {
                        this.downgradeToPrimary();
                    }
                }
                break;
                
            case 'I_AM_PRIMARY':
                // Another tab has claimed primary status
                if (data.instanceId !== this.instanceId) {
                    if (this.isPrimary) {
                        // Two primaries! Use instanceId to resolve
                        if (data.instanceId < this.instanceId) {
                            // They win, we downgrade
                            this.downgradeToPrimary();
                        }
                    } else {
                        // We're secondary, update last heartbeat
                        this.lastHeartbeat = Date.now();
                    }
                }
                break;
        }
    }
    
    /**
     * Initiate election to become primary
     */
    initiateElection() {
        if (!this.broadcastChannel) {
            this.becomePrimary();
            return;
        }
        
        // Broadcast our candidacy
        this.broadcastChannel.postMessage({
            type: 'ELECTION',
            instanceId: this.instanceId
        });
        
        // Wait to see if anyone with lower ID objects
        clearTimeout(this.electionTimer);
        this.electionTimer = setTimeout(() => {
            if (!this.isPrimary) {
                this.becomePrimary();
            }
        }, 500);
    }
    
    /**
     * Become the primary connection
     */
    becomePrimary() {
        if (this.isPrimary) return;
        
        this.isPrimary = true;
        this.connectionState = 'connected';
        
        // Announce that we are now primary
        if (this.broadcastChannel) {
            this.broadcastChannel.postMessage({
                type: 'I_AM_PRIMARY',
                instanceId: this.instanceId
            });
        }
        
        this.connectDirectly();
        
        // Start heartbeat
        this.heartbeatTimer = setInterval(() => {
            if (this.broadcastChannel && this.isPrimary) {
                this.broadcastChannel.postMessage({
                    type: 'HEARTBEAT',
                    instanceId: this.instanceId
                });
            }
        }, this.gs_heartbeatInterval());
        
        this.triggerCallback('primary');
    }
    
    /**
     * Become a secondary connection
     */
    becomeSecondary() {
        if (!this.isPrimary && this.eventSource) return;
        
        this.isPrimary = false;
        this.connectionState = 'connected';
        this.lastHeartbeat = Date.now();
        
        // Monitor for primary failure
        this.failoverTimer = setInterval(() => {
            const timeSinceLastHeartbeat = Date.now() - (this.lastHeartbeat || 0);
            if (timeSinceLastHeartbeat > this.gs_failoverCheckInterval()) {
                // Primary might be dead, initiate election
                this.initiateElection();
            }
        }, this.gs_failoverCheckInterval());
        
        this.triggerCallback('secondary');
        this.triggerCallback('open');
    }
    
    /**
     * Downgrade from primary to secondary
     */
    downgradeToPrimary() {
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
        
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
        
        this.isPrimary = false;
        this.becomeSecondary();
    }
    
    /**
     * Connect directly to SSE endpoint
     */
    connectDirectly() {
        // Close any existing connection first
        if (this.eventSource) {
            this.eventSource.onopen = null;
            this.eventSource.onmessage = null;
            this.eventSource.onerror = null;
            this.eventSource.close();
            this.eventSource = null;
        }
        
        this.eventSource = new EventSource(this.url);
        
        this.eventSource.onopen = () => {
            this.connectionState = 'connected';
            this.triggerCallback('open');
        };
        
        this.eventSource.onmessage = (event) => {
            // Trigger local callback
            this.triggerCallback('message', event.data);
            
            // Broadcast to other tabs if primary
            if (this.isPrimary && this.broadcastChannel) {
                this.broadcastChannel.postMessage({
                    type: 'MESSAGE',
                    message: event.data,
                    instanceId: this.instanceId
                });
            }
        };
        
        this.eventSource.onerror = (error) => {
            this.triggerCallback('error', error);
            
            if (this.eventSource.readyState === EventSource.CLOSED) {
                this.handleConnectionLost();
            }
        };
    }
    
    /**
     * Handle connection loss
     */
    handleConnectionLost() {
        this.connectionState = 'disconnected';
        
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
        
        this.triggerCallback('close');
        
        // Attempt reconnect if enabled
        if (this.gs_reconnect() && this.isPrimary) {
            this.reconnectTimer = setTimeout(() => {
                if (this.isPrimary) {
                    this.connectDirectly();
                }
            }, this.gs_reconnectInterval());
        }
    }
    
    /**
     * Clear all timers
     */
    clearTimers() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
        
        if (this.electionTimer) {
            clearTimeout(this.electionTimer);
            this.electionTimer = null;
        }
        
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        
        if (this.failoverTimer) {
            clearInterval(this.failoverTimer);
            this.failoverTimer = null;
        }
    }
    
    /**
     * Register event callback
     */
    on(event, callback) {
        if (this.callbacks[event]) {
            this.callbacks[event].push(callback);
        }
    }
    
    /**
     * Trigger event callbacks
     */
    triggerCallback(event, data) {
        if (this.callbacks[event]) {
            this.callbacks[event].forEach(callback => callback(data));
        }
    }
    
    /**
     * Check if this tab is the primary connection
     */
    isPrimaryConnection() {
        return this.isPrimary;
    }
    
    /**
     * Get current connection state
     */
    getConnectionState() {
        return this.connectionState;
    }
    
    /**
     * Get current state
     */
    getState() {
        const state = {};
        state["connectionState"] = this.connectionState;
        state["isPrimary"] = this.isPrimary;
        state["instanceId"] = this.instanceId;
        state["url"] = this.url;
        return state;
    }
}

// Export for ES module usage
export { SSEManager };
