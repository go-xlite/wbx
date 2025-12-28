/**
 * SSEManager - Public API wrapper
 * Provides intellisense and proxies to actual implementation
 * Uses numeric API indices to avoid minification issues
 */

// API indices (must match sse-manager-impl.js constructor)
const API = {
    CONNECT: 0,
    DISCONNECT: 1,
    ON: 2,
    IS_PRIMARY: 3,
    GET_STATE_CONN: 4,
    GET_STATE: 5,
    GS_RECONNECT: 6,
    GS_RECONNECT_INTERVAL: 7,
    GS_HEARTBEAT_INTERVAL: 8,
    GS_ELECTION_TIMEOUT: 9,
    GS_FAILOVER_CHECK_INTERVAL: 10
};



/**
 * @typedef {Object} SSEManagerCallbacks
 * @property {function} [open] - Called when connection opens
 * @property {function(MessageEvent)} [message] - Called when message received
 * @property {function(Event)} [error] - Called on error
 * @property {function} [close] - Called when connection closes
 * @property {function} [primary] - Called when this tab becomes primary
 * @property {function} [secondary] - Called when this tab becomes secondary
 */

/**
 * SSE Manager with BroadcastChannel coordination
 * Proxy wrapper that dynamically loads the actual implementation
 */
class SSEManager {
    /**
     * @param {string} url - The SSE endpoint URL
     */
    constructor(url) {
        this._ = null;  // Implementation instance (short name to survive minification)
        this.u = url;
    }

    async _init() {
        if (!this._) {
            const module = await import('./sse-manager-impl.js');
            this._ = new module.SSEManager(this.u);
        }
        return this._;
    }

    /**
     * Connect to SSE endpoint with coordination
     * @returns {Promise<void>}
     */
    async connect() {
        const impl = await this._init();
        return impl.$[API.CONNECT]();
    }

    /**
     * Disconnect from SSE endpoint
     * @returns {void}
     */
    disconnect() {
        this._.$[API.DISCONNECT]();
    }

    /**
     * Register event callback
     * @param {string} event - Event name ('open', 'message', 'error', 'close', 'primary', 'secondary')
     * @param {function} callback - Callback function
     * @returns {void}
     */
    on(event, callback) {
        this._.$[API.ON](event, callback);
    }

    /**
     * Check if this tab is the primary connection
     * @returns {boolean}
     */
    isPrimaryConnection() {
        return this._.$[API.IS_PRIMARY]();
    }

    /**
     * Get current connection state
     * @returns {'disconnected'|'connecting'|'connected'}
     */
    getConnectionState() {
        return this._.$[API.GET_STATE_CONN]();
    }

    /**
     * Get current state
     * @returns {Object}
     */
    getState() { return this._.$[API.GET_STATE](); }

    // Option setters (proxy to implementation)
    set reconnect(value) { this._.$[API.GS_RECONNECT](value); }
    set reconnectInterval(value) { this._.$[API.GS_RECONNECT_INTERVAL](value); }
    set heartbeatInterval(value) { this._.$[API.GS_HEARTBEAT_INTERVAL](value); }
    set electionTimeout(value) { this._.$[API.GS_ELECTION_TIMEOUT](value); }
    set failoverCheckInterval(value) { this._.$[API.GS_FAILOVER_CHECK_INTERVAL](value); }
    // Option getters (proxy to implementation)
    get reconnect() { return this._.$[API.GS_RECONNECT](); }
    get reconnectInterval() { return this._.$[API.GS_RECONNECT_INTERVAL](); }
    get heartbeatInterval() { return this._.$[API.GS_HEARTBEAT_INTERVAL](); }
    get electionTimeout() { return this._.$[API.GS_ELECTION_TIMEOUT](); }
    get failoverCheckInterval() { return this._.$[API.GS_FAILOVER_CHECK_INTERVAL](); }
}

/**
 * Create an SSEManager instance
 * @param {string} url - The SSE endpoint URL
 * @returns {Promise<SSEManager>} SSEManager instance with implementation loaded
 */
export async function createSSEManager(url) {
    const manager = new SSEManager(url);
    await manager._init();
    return manager;
}

/**
 * Preload the SSEManager implementation without creating an instance
 * @returns {Promise<typeof SSEManager>} Promise that resolves to SSEManager class
 */
export async function preloadSSEManager() {
    const module = await import('./sse-manager-impl.js');
    return module.SSEManager;
}

// Export stub class for type checking only
export { SSEManager };
