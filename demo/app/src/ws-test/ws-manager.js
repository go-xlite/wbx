/**
 * WebSocketManager - Public API wrapper
 * Provides intellisense and proxies to actual implementation
 */

let mod;

const WS_EVENT = {
    MESSAGE: 1,
    OPEN: 2,
    ERROR: 4,
    CLOSE: 8
}

const WS_STATE = {
    DISCONNECTED: 1,
    CONNECTING: 2,
    CONNECTED: 4,
    ERROR: 8
}

const WS_MODE = {
    WORKER: 1,
    DIRECT: 4
}

const WS_COORD_EVENT = {
    ENABLED: 1,
    BECAME_PRIMARY: 2,
    BECAME_SECONDARY: 4,
    TABS_UPDATED: 8
}

/**
 * WebSocket Manager with SharedWorker coordination
 * Proxy wrapper that dynamically loads the actual implementation
 */
class WebSocketManager {
    constructor() {
        this._ = null;  // Implementation instance (short name to survive minification)
        // these are only stubs for intellisense purposes
    }

    /**
     * Connect to WebSocket
     * @returns {void}
     */
    connect() {
        return this._.connect();
    }

    /**
     * Disconnect from WebSocket
     * @returns {void}
     */
    disconnect() {
        this._.disconnect();
    }

    /**
     * Send message through WebSocket
     * @param {string|object} message - Message to send
     * @returns {boolean} Success
     */
    send(message) {
        return this._.send(message);
    }

    /**
     * Register event callback
     * @param {number} event - Event type from WS_EVENT
     * @param {function} callback - Callback function
     * @returns {WebSocketManager} This instance for chaining
     */
    on(event, callback) {
        this._.on(event, callback);
        return this;
    }

    /**
     * Register coordination event callback
     * @param {number} event - Event type from WS_COORD_EVENT
     * @param {function} callback - Callback function
     * @returns {WebSocketManager} This instance for chaining
     */
    onCoordinationEvent(event, callback) {
        this._.onCoordinationEvent(event, callback);
        return this;
    }

    /**
     * Set preferred connection mode
     * @param {number} mode - Mode from WS_MODE
     * @returns {boolean} Success
     */
    setPreferredMode(mode) {
        return this._.setPreferredMode(mode);
    }

    /**
     * Clear preferred connection mode
     * @returns {void}
     */
    clearPreferredMode() {
        this._.clearPreferredMode();
    }

    /**
     * Connect via SharedWorker
     * @returns {void}
     */
    connectViaSharedWorker() {
        this._.connectViaSharedWorker();
    }

    /**
     * Connect directly
     * @returns {void}
     */
    connectDirectly() {
        this._.connectDirectly();
    }

    /**
     * Initialize connection coordination
     * @returns {boolean} Success
     */
    initConnectionCoordination() {
        return this._.initConnectionCoordination();
    }

    /**
     * Get all known tabs
     * @returns {Array} Known tabs
     */
    getKnownTabs() {
        return this._.getKnownTabs();
    }

    /**
     * Check if this tab is primary
     * @returns {boolean}
     */
    isPrimary() {
        return this._.isPrimary();
    }

    /**
     * Check if messages can be sent (either connected or has coordination)
     * @returns {boolean}
     */
    canSend() {
        return this._.connectionState === 4 || // WS_STATE.CONNECTED
               (this._.broadcastChannel && this._.broadcastChannel !== null);
    }

    /**
     * Check if coordination is enabled
     * @returns {boolean}
     */
    isCoordinationEnabled() {
        return this._.broadcastChannel && this._.broadcastChannel !== null;
    }

    /**
     * Switch to a different connection mode with automatic reconnection
     * @param {number} mode - Mode from WS_MODE
     * @returns {boolean} Success
     */
    switchMode(mode) {
        const wasConnected = this._.connectionState === 4; // WS_STATE.CONNECTED
        
        if (this._.setPreferredMode(mode)) {
            if (wasConnected) {
                // Disconnect, wait for cleanup, then reconnect
                setTimeout(() => {
                    this._.connect();
                }, 1000);
            }
            return true;
        }
        return false;
    }

    /**
     * Reset to auto-detect mode with automatic reconnection
     * @returns {void}
     */
    resetMode() {
        const wasConnected = this._.connectionState === 4; // WS_STATE.CONNECTED
        
        this._.clearPreferredMode();
        
        if (wasConnected) {
            this._.disconnect();
            setTimeout(() => {
                this._.connect();
            }, 500);
        }
    }

    /**
     * Get human-readable mode name
     * @param {number} [mode] - Mode from WS_MODE, defaults to current mode
     * @returns {string}
     */
    getModeName(mode) {
        const m = mode || this._.connectionMode;
        const names = {
            [WS_MODE.WORKER]: 'SharedWorker',
            [WS_MODE.DIRECT]: 'Direct'
        };
        return names[m] || 'Unknown';
    }

    /**
     * Get debug/status information
     * @returns {Object} Debug information
     */
    getDebugInfo() {
        return {
            connectionId: this._.connectionId,
            connectionState: this._.connectionState,
            connectionMode: this._.connectionMode,
            modeName: this.getModeName(),
            isPrimary: this.isPrimary(),
            coordinationEnabled: this.isCoordinationEnabled(),
            canSend: this.canSend(),
            knownTabs: this._.getKnownTabs()
        };
    }

    /**
     * Get coordination information
     * @returns {Object} Coordination details
     */
    getCoordinationInfo() {
        return {
            enabled: this.isCoordinationEnabled(),
            isPrimary: this.isPrimary(),
            tabs: this._.getKnownTabs()
        };
    }

    /**
     * Get connection state
     * @returns {number} State from WS_STATE
     */
    get connectionState() {
        return this._.connectionState;
    }

    /**
     * Get connection mode
     * @returns {number} Mode from WS_MODE
     */
    get connectionMode() {
        return this._.connectionMode;
    }

    /**
     * Get connection ID
     * @returns {string}
     */
    get connectionId() {
        return this._.connectionId;
    }

    /**
     * Get broadcast channel
     * @returns {BroadcastChannel}
     */
    get broadcastChannel() {
        return this._.broadcastChannel;
    }
}

/**
 * Create a WebSocketManager instance
 * @param {Object} [options] - Configuration options
 * @returns {Promise<WebSocketManager>} WebSocketManager instance with implementation loaded
 */
const importPath = '../../ws/manager.js';
export async function createWebSocketManager(options) {
    if (!mod) {
        mod = await import(importPath);
    }
    const manager = new WebSocketManager();
    manager._ = await mod.createWebSocketManager(options);
    return manager;
}

// Export stub class for type checking only
export { WebSocketManager, WS_EVENT, WS_STATE, WS_MODE, WS_COORD_EVENT };
