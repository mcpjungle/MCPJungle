// Advanced polling system for real-time updates
class PollingManager {
    constructor() {
        this.pollers = new Map();
        this.globalSettings = {
            defaultInterval: 30000, // 30 seconds
            maxRetries: 3,
            backoffMultiplier: 2,
            maxBackoffTime: 300000, // 5 minutes
            pauseWhenHidden: true
        };
        this.isPageVisible = !document.hidden;
        this.setupVisibilityHandling();
    }

    setupVisibilityHandling() {
        document.addEventListener('visibilitychange', () => {
            this.isPageVisible = !document.hidden;

            if (this.globalSettings.pauseWhenHidden) {
                if (this.isPageVisible) {
                    this.resumeAll();
                } else {
                    this.pauseAll();
                }
            }
        });
    }

    // Create a new poller
    create(id, config = {}) {
        const poller = new Poller(id, {
            ...this.globalSettings,
            ...config
        });

        this.pollers.set(id, poller);
        return poller;
    }

    // Get existing poller
    get(id) {
        return this.pollers.get(id);
    }

    // Remove poller
    remove(id) {
        const poller = this.pollers.get(id);
        if (poller) {
            poller.stop();
            this.pollers.delete(id);
        }
    }

    // Pause all pollers
    pauseAll() {
        this.pollers.forEach(poller => poller.pause());
    }

    // Resume all pollers
    resumeAll() {
        this.pollers.forEach(poller => poller.resume());
    }

    // Stop all pollers
    stopAll() {
        this.pollers.forEach(poller => poller.stop());
        this.pollers.clear();
    }

    // Get status of all pollers
    getStatus() {
        const status = {};
        this.pollers.forEach((poller, id) => {
            status[id] = poller.getStatus();
        });
        return status;
    }
}

class Poller {
    constructor(id, config) {
        this.id = id;
        this.config = config;
        this.state = {
            isRunning: false,
            isPaused: false,
            currentInterval: config.interval || config.defaultInterval,
            retryCount: 0,
            lastSuccess: null,
            lastError: null,
            consecutiveErrors: 0
        };
        this.timeoutId = null;
        this.callbacks = {
            success: [],
            error: [],
            retry: [],
            statusChange: []
        };
    }

    // Start polling
    start() {
        if (this.state.isRunning) return this;

        this.state.isRunning = true;
        this.state.isPaused = false;
        this.emit('statusChange', { status: 'started' });
        this.scheduleNext();
        return this;
    }

    // Stop polling
    stop() {
        if (this.timeoutId) {
            clearTimeout(this.timeoutId);
            this.timeoutId = null;
        }

        this.state.isRunning = false;
        this.state.isPaused = false;
        this.emit('statusChange', { status: 'stopped' });
        return this;
    }

    // Pause polling
    pause() {
        if (!this.state.isRunning) return this;

        if (this.timeoutId) {
            clearTimeout(this.timeoutId);
            this.timeoutId = null;
        }

        this.state.isPaused = true;
        this.emit('statusChange', { status: 'paused' });
        return this;
    }

    // Resume polling
    resume() {
        if (!this.state.isRunning || !this.state.isPaused) return this;

        this.state.isPaused = false;
        this.emit('statusChange', { status: 'resumed' });
        this.scheduleNext();
        return this;
    }

    // Execute poll immediately
    async poll() {
        if (!this.config.pollFunction) {
            throw new Error('No poll function configured');
        }

        try {
            const result = await this.config.pollFunction();
            this.handleSuccess(result);
            return result;
        } catch (error) {
            this.handleError(error);
            throw error;
        }
    }

    // Schedule next poll
    scheduleNext() {
        if (!this.state.isRunning || this.state.isPaused) return;

        this.timeoutId = setTimeout(async () => {
            try {
                await this.poll();
            } catch (error) {
                // Error already handled in poll()
            }

            // Schedule next poll if still running
            if (this.state.isRunning && !this.state.isPaused) {
                this.scheduleNext();
            }
        }, this.state.currentInterval);
    }

    // Handle successful poll
    handleSuccess(result) {
        this.state.lastSuccess = new Date();
        this.state.lastError = null;
        this.state.retryCount = 0;
        this.state.consecutiveErrors = 0;
        this.state.currentInterval = this.config.interval || this.config.defaultInterval;

        this.emit('success', result);
    }

    // Handle poll error
    handleError(error) {
        this.state.lastError = error;
        this.state.consecutiveErrors++;

        // Implement exponential backoff
        if (this.state.consecutiveErrors > 1) {
            this.state.currentInterval = Math.min(
                this.state.currentInterval * this.config.backoffMultiplier,
                this.config.maxBackoffTime
            );
        }

        // Check if we should retry
        if (this.state.retryCount < this.config.maxRetries) {
            this.state.retryCount++;
            this.emit('retry', {
                error,
                retryCount: this.state.retryCount,
                nextInterval: this.state.currentInterval
            });
        } else {
            this.emit('error', error);
        }
    }

    // Add event listener
    on(event, callback) {
        if (!this.callbacks[event]) {
            this.callbacks[event] = [];
        }
        this.callbacks[event].push(callback);
        return this;
    }

    // Remove event listener
    off(event, callback) {
        if (this.callbacks[event]) {
            const index = this.callbacks[event].indexOf(callback);
            if (index > -1) {
                this.callbacks[event].splice(index, 1);
            }
        }
        return this;
    }

    // Emit event
    emit(event, data) {
        if (this.callbacks[event]) {
            this.callbacks[event].forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`Error in ${event} callback:`, error);
                }
            });
        }
    }

    // Get poller status
    getStatus() {
        return {
            id: this.id,
            isRunning: this.state.isRunning,
            isPaused: this.state.isPaused,
            currentInterval: this.state.currentInterval,
            retryCount: this.state.retryCount,
            lastSuccess: this.state.lastSuccess,
            lastError: this.state.lastError,
            consecutiveErrors: this.state.consecutiveErrors
        };
    }

    // Update configuration
    updateConfig(newConfig) {
        this.config = { ...this.config, ...newConfig };
        return this;
    }
}

// Smart polling utility for common use cases
class SmartPoller {
    constructor(pollingManager) {
        this.pm = pollingManager;
    }

    // Create a poller for API endpoints
    createApiPoller(id, endpoint, options = {}) {
        const config = {
            interval: options.interval || 30000,
            pollFunction: async () => {
                const response = await fetch(endpoint, {
                    method: options.method || 'GET',
                    headers: options.headers || {},
                    ...options.fetchOptions
                });

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                return response.json();
            },
            ...options
        };

        return this.pm.create(id, config);
    }

    // Create a poller with data comparison
    createDataPoller(id, pollFunction, options = {}) {
        let lastData = null;
        let lastDataHash = null;

        const config = {
            interval: options.interval || 30000,
            pollFunction: async () => {
                const data = await pollFunction();
                const dataHash = this.hashData(data);

                // Only emit if data changed
                if (dataHash !== lastDataHash) {
                    const previousData = lastData;
                    lastData = data;
                    lastDataHash = dataHash;

                    return {
                        data,
                        previousData,
                        hasChanged: true
                    };
                }

                return {
                    data,
                    previousData: lastData,
                    hasChanged: false
                };
            },
            ...options
        };

        return this.pm.create(id, config);
    }

    // Simple hash function for data comparison
    hashData(data) {
        return JSON.stringify(data).split('').reduce((hash, char) => {
            return ((hash << 5) - hash) + char.charCodeAt(0);
        }, 0);
    }
}

// Connection status poller
class ConnectionPoller {
    constructor(pollingManager) {
        this.pm = pollingManager;
        this.status = 'unknown';
        this.callbacks = [];
    }

    start() {
        const poller = this.pm.create('connection-status', {
            interval: 15000, // Check every 15 seconds
            pollFunction: async () => {
                try {
                    const response = await fetch('/health', {
                        method: 'GET',
                        cache: 'no-cache',
                        signal: AbortSignal.timeout(5000)
                    });

                    if (response.ok) {
                        this.updateStatus('connected');
                        return { status: 'connected', timestamp: new Date() };
                    } else {
                        throw new Error(`HTTP ${response.status}`);
                    }
                } catch (error) {
                    this.updateStatus('disconnected');
                    throw error;
                }
            }
        });

        poller.on('success', (result) => {
            this.updateStatus('connected');
        });

        poller.on('error', (error) => {
            this.updateStatus('disconnected');
        });

        poller.start();
        return poller;
    }

    updateStatus(newStatus) {
        if (this.status !== newStatus) {
            const oldStatus = this.status;
            this.status = newStatus;

            this.callbacks.forEach(callback => {
                try {
                    callback(newStatus, oldStatus);
                } catch (error) {
                    console.error('Error in connection status callback:', error);
                }
            });
        }
    }

    onStatusChange(callback) {
        this.callbacks.push(callback);
    }

    getStatus() {
        return this.status;
    }
}

// Global polling manager instance
const globalPollingManager = new PollingManager();
const smartPoller = new SmartPoller(globalPollingManager);
const connectionPoller = new ConnectionPoller(globalPollingManager);

// Make available globally
window.PollingManager = PollingManager;
window.Poller = Poller;
window.SmartPoller = SmartPoller;
window.ConnectionPoller = ConnectionPoller;
window.globalPollingManager = globalPollingManager;
window.smartPoller = smartPoller;
window.connectionPoller = connectionPoller;

// Auto-start connection monitoring
document.addEventListener('DOMContentLoaded', () => {
    connectionPoller.start();
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    globalPollingManager.stopAll();
});