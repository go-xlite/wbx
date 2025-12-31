import { pathPrefix } from "../../../lib/site_core";
/**
 * SSEManager - Public API wrapper
 * Provides intellisense and proxies to actual implementation
 */

let mod;


const SSE_EVENT = {
    MESSAGE: 0,
    OPEN: 1,
    ERROR: 2,
    CLOSE: 3,
    PRIMARY: 4,
    SECONDARY: 5
}

const SSE_STATE = {
    DISCONNECTED: 0,
    CONNECTING: 1,
    CONNECTED: 2
}

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
        this.ops = {
            reconnect: true,
            reconnectInterval: 5000,
            heartbeatInterval: 2000,
            electionTimeout: 500,
            failoverCheckInterval: 10000
        };
        // these are only stubs for intellisense purposes
    }



    /**
     * Connect to SSE endpoint with coordination
     * @returns {Promise<void>}
     */
    async connect() {
        return this._.connect();
    }

    /**
     * Disconnect from SSE endpoint
     * @returns {void}
     */
    disconnect() {
        this._.disconnect();
    }

    /**
     * Register event callback
     * @param {number} event - Event type from SSE_EVENT (MESSAGE, OPEN, ERROR, CLOSE, PRIMARY, SECONDARY)
     * @param {function} callback - Callback function
     * @returns {void}
     */
    on(event, callback) {
        this._.on(event, callback);
    }

    /**
     * Check if this tab is the primary connection
     * @returns {boolean}
     */
    isPrimaryConnection() {
        return this._.isPrimaryConnection();
    }

    /**
     * Get current connection state
     * @returns {number} State from SSE_STATE (DISCONNECTED, CONNECTING, CONNECTED)
     */
    getConnectionState() {
        return this._.getConnectionState();
    }

    /**
     * Get current state
     * @returns {Object}
     */
    getState() { return this._.getState(); }
}



/**
 * Create an SSEManager instance
 * @param {string} url - The SSE endpoint URL
 * @param {SSEManagerOptions} [options] - Configuration options
 * @returns {Promise<SSEManager>} SSEManager instance with implementation loaded
 */
const importPath = pathPrefix+'/sse/p/sse-manager-impl.js';
export async function createSSEManager(url, options) {
    if (!mod) {
        mod = await import(importPath);
    }
    const manager = new SSEManager(url);
    manager._ = new mod.SSEManager(manager.u, options);
    manager.ops = manager._.ops;
    return manager;
}



// Export stub class for type checking only
export { SSEManager, SSE_EVENT, SSE_STATE };
