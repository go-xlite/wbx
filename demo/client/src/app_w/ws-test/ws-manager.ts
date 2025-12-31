/**
 * WebSocketManager - Public API wrapper
 * Provides type-safe interface for WebSocket connections with SharedWorker coordination
 */

import { pathPrefix } from '../../../lib/site_core.js';

// ==================== Type Definitions ====================

export interface WsManagerOptions {
    debug?: boolean;
    autoConnect?: boolean;
    reconnectOnDisconnect?: boolean;
    maxReconnectAttempts?: number;
    endpoint?: string;
    sessionStrategy?: number;
    wsRoute?: string;
    wsWorkerRoute?: string;
}

export interface WsMessage {
    type: string;
    clientID: string;
    sessionID: string;
    userID: number;
    username: string;
    message: string;
    timestamp: number;
}

export interface TabInfo {
    id: string;
    isPrimary: boolean;
    lastHeartbeat: number;
}

export interface DebugInfo {
    connectionId: string;
    sessionId: string;
    connectionState: number;
    connectionMode: number;
    modeName: string;
    isPrimary: boolean;
    coordinationEnabled: boolean;
    canSend: boolean;
    knownTabs: TabInfo[];
    sessionData: Record<string, any>;
}

export interface CoordinationInfo {
    enabled: boolean;
    isPrimary: boolean;
    tabs: TabInfo[];
}

/**
 * WebSocket Manager Interface
 * Defines the public API for WebSocket connections with SharedWorker coordination
 */
export interface IWebSocketManager {
    /** Connect to WebSocket */
    connect(): void;
    
    /** Disconnect from WebSocket */
    disconnect(): void;
    
    /** Send message through WebSocket */
    send(message: string | object): boolean;
    
    /** Register event callback for connection events (MESSAGE, OPEN, ERROR, CLOSE) */
    on(event: WsEvent, callback: (data?: any) => void): IWebSocketManager;
    
    /** Register coordination event callback (ENABLED, BECAME_PRIMARY, BECAME_SECONDARY, TABS_UPDATED) */
    onCoordinationEvent(event: WsCoordEvent, callback: (data?: any) => void): IWebSocketManager;
    
    /** Set preferred connection mode (WORKER or DIRECT) */
    setPreferredMode(mode: WsMode): boolean;
    
    /** Clear preferred connection mode (auto-detect) */
    clearPreferredMode(): void;
    
    /** Connect via SharedWorker */
    connectViaSharedWorker(): void;
    
    /** Connect directly without SharedWorker */
    connectDirectly(): void;
    
    /** Initialize connection coordination for multi-tab support */
    initConnectionCoordination(): boolean;
    
    /** Get all known tabs in the coordination group */
    getKnownTabs(): TabInfo[];
    
    /** Check if this tab is the primary connection */
    isPrimary(): boolean;
    
    /** Check if messages can be sent (connected or coordinated) */
    canSend(): boolean;
    
    /** Check if tab coordination is enabled */
    isCoordinationEnabled(): boolean;
    
    /** Switch to a different connection mode with automatic reconnection */
    switchMode(mode: WsMode): boolean;
    
    /** Reset to auto-detect mode with automatic reconnection */
    resetMode(): void;
    
    /** Get human-readable name for connection mode */
    getModeName(mode?: WsMode): string;
    
    /** Get comprehensive debug/status information */
    getDebugInfo(): DebugInfo;
    
    /** Get tab coordination information */
    getCoordinationInfo(): CoordinationInfo;
    
    /** Set a value in the session storage */
    setSessionValue(key: string, value: any): void;
    
    /** Get a value from the session storage */
    getSessionValue(key: string): any;
    
    /** Delete a value from the session storage */
    deleteSessionValue(key: string): void;
    
    /** Clear all session data */
    clearSession(): void;
    
    // Read-only properties
    readonly connectionState: WsState;
    readonly connectionMode: WsMode | null;
    readonly connectionId: string;
    readonly sessionId: string;
    readonly sessionStrategy: WsSessionStrategy;
    readonly sessionData: Record<string, any>;
    readonly broadcastChannel: BroadcastChannel | null;
}

// ==================== Constants ====================

export const WS_EVENT = {
    MESSAGE: 1,
    OPEN: 2,
    ERROR: 4,
    CLOSE: 8
} as const;

export const WS_STATE = {
    DISCONNECTED: 1,
    CONNECTING: 2,
    CONNECTED: 4,
    ERROR: 8
} as const;

export const WS_MODE = {
    WORKER: 1,
    DIRECT: 4
} as const;

export const WS_COORD_EVENT = {
    ENABLED: 1,
    BECAME_PRIMARY: 2,
    BECAME_SECONDARY: 4,
    TABS_UPDATED: 8
} as const;

export const WS_SESSION_STRATEGY = {
    ISOLATED: 1,
    SHARED: 2,
    SHARED_CONNECTION: 4
} as const;

export type WsEvent = typeof WS_EVENT[keyof typeof WS_EVENT];
export type WsState = typeof WS_STATE[keyof typeof WS_STATE];
export type WsMode = typeof WS_MODE[keyof typeof WS_MODE];
export type WsCoordEvent = typeof WS_COORD_EVENT[keyof typeof WS_COORD_EVENT];
export type WsSessionStrategy = typeof WS_SESSION_STRATEGY[keyof typeof WS_SESSION_STRATEGY];

// ==================== Implementation ====================

let mod: any;

export class WebSocketManager implements IWebSocketManager {
    private _: any;

    constructor() {
        this._ = null;
    }

    connect(): void {
        return this._.connect();
    }

    disconnect(): void {
        this._.disconnect();
    }

    send(message: string | object): boolean {
        return this._.send(message);
    }

    on(event: WsEvent, callback: (data?: any) => void): WebSocketManager {
        this._.on(event, callback);
        return this;
    }

    onCoordinationEvent(event: WsCoordEvent, callback: (data?: any) => void): WebSocketManager {
        this._.onCoordinationEvent(event, callback);
        return this;
    }

    setPreferredMode(mode: WsMode): boolean {
        return this._.setPreferredMode(mode);
    }

    clearPreferredMode(): void {
        this._.clearPreferredMode();
    }

    connectViaSharedWorker(): void {
        this._.connectViaSharedWorker();
    }

    connectDirectly(): void {
        this._.connectDirectly();
    }

    initConnectionCoordination(): boolean {
        return this._.initConnectionCoordination();
    }

    getKnownTabs(): TabInfo[] {
        return this._.getKnownTabs();
    }

    isPrimary(): boolean {
        return this._.isPrimary();
    }

    canSend(): boolean {
        return this._.connectionState === WS_STATE.CONNECTED ||
               (this._.broadcastChannel && this._.broadcastChannel !== null);
    }

    isCoordinationEnabled(): boolean {
        return this._.broadcastChannel && this._.broadcastChannel !== null;
    }

    switchMode(mode: WsMode): boolean {
        const wasConnected = this._.connectionState === WS_STATE.CONNECTED;
        
        if (this._.setPreferredMode(mode)) {
            if (wasConnected) {
                setTimeout(() => {
                    this._.connect();
                }, 1000);
            }
            return true;
        }
        return false;
    }

    resetMode(): void {
        const wasConnected = this._.connectionState === WS_STATE.CONNECTED;
        
        this._.clearPreferredMode();
        
        if (wasConnected) {
            this._.disconnect();
            setTimeout(() => {
                this._.connect();
            }, 500);
        }
    }

    getModeName(mode?: WsMode): string {
        const m = mode || this._.connectionMode;
        const names: Record<number, string> = {
            [WS_MODE.WORKER]: 'SharedWorker',
            [WS_MODE.DIRECT]: 'Direct'
        };
        return names[m] || 'Unknown';
    }

    getDebugInfo(): DebugInfo {
        return {
            connectionId: this._.connectionId,
            sessionId: this._.sessionId,
            connectionState: this._.connectionState,
            connectionMode: this._.connectionMode,
            modeName: this.getModeName(),
            isPrimary: this.isPrimary(),
            coordinationEnabled: this.isCoordinationEnabled(),
            canSend: this.canSend(),
            knownTabs: this._.getKnownTabs(),
            sessionData: this._.sessionData
        };
    }

    getCoordinationInfo(): CoordinationInfo {
        return {
            enabled: this.isCoordinationEnabled(),
            isPrimary: this.isPrimary(),
            tabs: this._.getKnownTabs()
        };
    }

    get connectionState(): WsState {
        return this._.connectionState;
    }

    get connectionMode(): WsMode | null {
        return this._.connectionMode;
    }

    get connectionId(): string {
        return this._.connectionId;
    }

    get sessionId(): string {
        return this._.sessionId;
    }

    get sessionStrategy(): WsSessionStrategy {
        return this._.sessionStrategy;
    }

    get sessionData(): Record<string, any> {
        return this._.sessionData;
    }

    setSessionValue(key: string, value: any): void {
        this._.setSessionValue(key, value);
    }

    getSessionValue(key: string): any {
        return this._.getSessionValue(key);
    }

    deleteSessionValue(key: string): void {
        this._.deleteSessionValue(key);
    }

    clearSession(): void {
        this._.clearSession();
    }

    get broadcastChannel(): BroadcastChannel | null {
        return this._.broadcastChannel;
    }
}


/**
 * Create a WebSocketManager instance
 * @param options - Configuration options
 * @returns WebSocketManager instance with implementation loaded
 */
export async function createWebSocketManager(options?: WsManagerOptions): Promise<WebSocketManager> {
    const endpoint = '/s/' + pathPrefix.split('/').pop();
    const ixp = '/m/xlite/ws/p/'+'ws-manager-impl.js';
    
    if (!mod) {
        mod = await import(ixp);
    }

    const config: WsManagerOptions = {
        ...options,
        wsRoute: `${endpoint}/ws/connect`,
        wsWorkerRoute: `/m/xlite/ws/p/ws-shared-worker.js?endpoint=${encodeURIComponent(endpoint + '/ws/connect')}`,
        endpoint: endpoint + '/ws/connect'
    };

    const manager = new WebSocketManager();
    manager['_'] = await mod.createWebSocketManager(config);
    return manager;
}
