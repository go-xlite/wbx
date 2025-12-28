/**
 * SSEManager Stub - Provides intellisense and dynamically imports the actual implementation
 * @file sse-manager-stub.js
 */

/**
 * @typedef {Object} SSEManagerOptions
 * @property {number} [heartbeatInterval=2000] - Interval for heartbeat messages (ms)
 * @property {number} [failoverCheckInterval=5000] - Interval to check for primary failure (ms)
 * @property {number} [electionTimeout=1000] - Timeout before initiating election (ms)
 * @property {boolean} [reconnect=true] - Whether to reconnect on error
 */

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
 * Ensures only one SSE connection across all tabs
 */
class SSEManager {
    /**
     * @param {string} url - The SSE endpoint URL
     * @param {SSEManagerOptions} [options] - Configuration options
     */
    constructor(url, options = {}) {
        throw new Error('SSEManager stub - use createSSEManager() to load the actual implementation');
    }

    /**
     * Connect to SSE endpoint with coordination
     * @returns {void}
     */
    connect() {}

    /**
     * Disconnect from SSE endpoint
     * @returns {void}
     */
    disconnect() {}

    /**
     * Register event callback
     * @param {string} event - Event name ('open', 'message', 'error', 'close', 'primary', 'secondary')
     * @param {function} callback - Callback function
     * @returns {void}
     */
    on(event, callback) {}

    /**
     * Check if this tab is the primary connection
     * @returns {boolean}
     */
    isPrimaryConnection() {
        return false;
    }

    /**
     * Get current connection state
     * @returns {'disconnected'|'connecting'|'connected'}
     */
    getConnectionState() {
        return 'disconnected';
    }
}

/**
 * Dynamically load and create an SSEManager instance
 * @param {string} url - The SSE endpoint URL
 * @param {SSEManagerOptions} [options] - Configuration options
 * @returns {Promise<SSEManager>} Promise that resolves to SSEManager instance
 */
export async function createSSEManager(url, options = {}) {
    const module = await import('./sse-manager.js');
    return new module.SSEManager(url, options);
}

/**
 * Preload the SSEManager module without creating an instance
 * @returns {Promise<typeof SSEManager>} Promise that resolves to SSEManager class
 */
export async function preloadSSEManager() {
    const module = await import('./sse-manager.js');
    return module.SSEManager;
}

// Export stub class for type checking only
export { SSEManager };
