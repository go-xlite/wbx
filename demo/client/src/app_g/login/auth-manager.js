/**
 * AuthManager - Client-side authentication manager
 * Handles login, logout, registration, and session management
 */
export class AuthManager {
    constructor(options = {}) {
        this.baseUrl = window.PATH_PREFIX+'/auth';
        this.onAuthChange = options.onAuthChange || null;
    }

    /**
     * Login with username and password
     * @param {string} username 
     * @param {string} password 
     * @returns {Promise<{success: boolean, username?: string, role?: string, error?: string}>}
     */
    async login(username, password) {
        try {
            const response = await fetch(`${this.baseUrl}/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                credentials: 'include', // Important: include cookies
                body: JSON.stringify({ username, password }),
            });

            const data = await response.json();

            if (response.ok && data.success) {
                this._notifyAuthChange(true, data);
                return { success: true, username: data.username, role: data.role };
            }

            return { success: false, error: data.error || 'Login failed' };
        } catch (err) {
            return { success: false, error: err.message || 'Network error' };
        }
    }

    /**
     * Logout the current user
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    async logout() {
        try {
            const response = await fetch(`${this.baseUrl}/logout`, {
                method: 'POST',
                credentials: 'include',
            });

            const data = await response.json();

            if (response.ok) {
                this._notifyAuthChange(false, null);
                return { success: true };
            }

            return { success: false, error: data.error || 'Logout failed' };
        } catch (err) {
            return { success: false, error: err.message || 'Network error' };
        }
    }

    /**
     * Register a new user
     * @param {string} username 
     * @param {string} password 
     * @param {string} role 
     * @returns {Promise<{success: boolean, username?: string, error?: string}>}
     */
    async register(username, password, role = 'user') {
        try {
            const response = await fetch(`${this.baseUrl}/register`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                credentials: 'include',
                body: JSON.stringify({ username, password, role }),
            });

            const data = await response.json();

            if (response.ok && data.success) {
                return { success: true, username: data.username, role: data.role };
            }

            return { success: false, error: data.error || 'Registration failed' };
        } catch (err) {
            return { success: false, error: err.message || 'Network error' };
        }
    }

    /**
     * Refresh the session token
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    async refresh() {
        try {
            const response = await fetch(`${this.baseUrl}/refresh`, {
                method: 'POST',
                credentials: 'include',
            });

            const data = await response.json();

            if (response.ok) {
                return { success: true };
            }

            return { success: false, error: data.error || 'Refresh failed' };
        } catch (err) {
            return { success: false, error: err.message || 'Network error' };
        }
    }

    /**
     * Check if user is authenticated by trying to access a protected endpoint
     * @returns {Promise<{authenticated: boolean, session?: object}>}
     */
    async checkAuth() {
        try {
            const response = await fetch(`${this.baseUrl}/me`, {
                method: 'GET',
                credentials: 'include',
            });

            if (response.ok) {
                const data = await response.json();
                return { authenticated: true, session: data.session };
            }

            return { authenticated: false };
        } catch (err) {
            return { authenticated: false };
        }
    }

    /**
     * Notify listeners of auth state change
     * @private
     */
    _notifyAuthChange(authenticated, userData) {
        if (this.onAuthChange) {
            this.onAuthChange(authenticated, userData);
        }
    }
}

// Export singleton instance for convenience
export const auth = new AuthManager();
