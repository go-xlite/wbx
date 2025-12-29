import { pathPrefix } from '../../lib/site_core.js';

/**
 * API Manager for fetching server instance data
 */
class APIManager {
    constructor() {
        // Get pathPrefix dynamically at runtime
        const segments = window.location.pathname.split('/').filter(s => s);
        const prefix = segments.length > 0 ? '/' + segments[0] : '';
        this.baseUrl = `${prefix}/trail`;
        this.loading = false;
        this.error = null;
    }

    /**
     * Fetch server instance list
     * @returns {Promise<Array>} Array of server instances
     */
    async fetchServerList() {
        this.loading = true;
        this.error = null;

        try {
            const response = await fetch(`${this.baseUrl}/servers/a/list`);
            
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const data = await response.json();
            this.loading = false;
            return data;
        } catch (error) {
            this.loading = false;
            this.error = error.message;
            throw error;
        }
    }

    /**
     * Get full server details (example for future expansion)
     * @param {string} serverId - The server ID
     * @returns {Promise<Object>} Server details
     */
    async fetchServerDetails(serverId) {
        this.loading = true;
        this.error = null;

        try {
            const response = await fetch(`${this.baseUrl}/servers/i/${serverId}/details`);
            
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const data = await response.json();
            this.loading = false;
            return data;
        } catch (error) {
            this.loading = false;
            this.error = error.message;
            throw error;
        }
    }

    /**
     * Fetch available filter options
     * @returns {Promise<Object>} Filter options (regions, zones, states, instance types)
     */
    async fetchFilters() {
        this.loading = true;
        this.error = null;

        try {
            const response = await fetch(`${this.baseUrl}/servers/a/filters`);
            
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const data = await response.json();
            this.loading = false;
            return data;
        } catch (error) {
            this.loading = false;
            this.error = error.message;
            throw error;
        }
    }
}

export function createAPIManager() {
    return new APIManager();
}
