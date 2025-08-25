// Offline detection and fallback system
class OfflineManager {
    constructor() {
        this.isOnline = navigator.onLine;
        this.offlineQueue = [];
        this.offlineData = new Map();
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupOfflineIndicator();
        this.setupServiceWorker();
        this.loadOfflineData();
    }

    setupEventListeners() {
        window.addEventListener('online', () => {
            this.handleOnline();
        });

        window.addEventListener('offline', () => {
            this.handleOffline();
        });

        // Intercept failed requests
        this.interceptFailedRequests();
    }

    handleOnline() {
        this.isOnline = true;
        this.updateOfflineIndicator();
        this.processOfflineQueue();
        this.showNotification('Connection restored', 'success');
    }

    handleOffline() {
        this.isOnline = false;
        this.updateOfflineIndicator();
        this.showNotification('You are now offline. Some features may be limited.', 'warning');
    }

    setupOfflineIndicator() {
        // Create offline indicator
        const indicator = document.createElement('div');
        indicator.id = 'offline-indicator';
        indicator.className = 'fixed bottom-4 left-4 z-50 px-4 py-2 rounded-lg shadow-lg transition-all duration-300 transform translate-y-full';
        indicator.innerHTML = `
            <div class="flex items-center space-x-2">
                <div class="w-2 h-2 rounded-full bg-red-500"></div>
                <span class="text-sm font-medium text-white">Offline</span>
            </div>
        `;
        document.body.appendChild(indicator);

        this.offlineIndicator = indicator;
        this.updateOfflineIndicator();
    }

    updateOfflineIndicator() {
        if (!this.offlineIndicator) return;

        if (this.isOnline) {
            this.offlineIndicator.classList.add('translate-y-full');
        } else {
            this.offlineIndicator.classList.remove('translate-y-full');
            this.offlineIndicator.className = this.offlineIndicator.className.replace(/bg-\w+-\d+/, 'bg-red-600');
        }
    }

    // Queue failed requests for retry when online
    interceptFailedRequests() {
        const originalFetch = window.fetch;

        window.fetch = async (url, options = {}) => {
            try {
                const response = await originalFetch(url, options);

                if (!response.ok && !this.isOnline) {
                    // Queue the request for retry
                    this.queueRequest(url, options);
                }

                return response;
            } catch (error) {
                if (!this.isOnline) {
                    this.queueRequest(url, options);

                    // Try to return cached data if available
                    const cachedData = this.getCachedData(url);
                    if (cachedData) {
                        return new Response(JSON.stringify(cachedData), {
                            status: 200,
                            headers: { 'Content-Type': 'application/json' }
                        });
                    }
                }

                throw error;
            }
        };
    }

    queueRequest(url, options) {
        this.offlineQueue.push({
            url,
            options,
            timestamp: Date.now()
        });
    }

    async processOfflineQueue() {
        if (this.offlineQueue.length === 0) return;

        const queue = [...this.offlineQueue];
        this.offlineQueue = [];

        for (const request of queue) {
            try {
                await fetch(request.url, request.options);
            } catch (error) {
                // Re-queue if still failing
                this.offlineQueue.push(request);
            }
        }

        if (this.offlineQueue.length > 0) {
            this.showNotification(`${this.offlineQueue.length} requests are still pending`, 'info');
        }
    }

    // Cache management for offline data
    cacheData(key, data) {
        this.offlineData.set(key, {
            data,
            timestamp: Date.now()
        });

        // Persist to localStorage
        try {
            localStorage.setItem(`offline_${key}`, JSON.stringify({
                data,
                timestamp: Date.now()
            }));
        } catch (error) {
            console.warn('Failed to cache data to localStorage:', error);
        }
    }

    getCachedData(key) {
        // Check memory cache first
        const memoryCache = this.offlineData.get(key);
        if (memoryCache && this.isCacheValid(memoryCache.timestamp)) {
            return memoryCache.data;
        }

        // Check localStorage
        try {
            const stored = localStorage.getItem(`offline_${key}`);
            if (stored) {
                const parsed = JSON.parse(stored);
                if (this.isCacheValid(parsed.timestamp)) {
                    return parsed.data;
                }
            }
        } catch (error) {
            console.warn('Failed to retrieve cached data:', error);
        }

        return null;
    }

    isCacheValid(timestamp, maxAge = 5 * 60 * 1000) { // 5 minutes default
        return Date.now() - timestamp < maxAge;
    }

    loadOfflineData() {
        // Load cached data from localStorage on startup
        try {
            for (let i = 0; i < localStorage.length; i++) {
                const key = localStorage.key(i);
                if (key && key.startsWith('offline_')) {
                    const data = localStorage.getItem(key);
                    if (data) {
                        const parsed = JSON.parse(data);
                        const cacheKey = key.replace('offline_', '');
                        this.offlineData.set(cacheKey, parsed);
                    }
                }
            }
        } catch (error) {
            console.warn('Failed to load offline data:', error);
        }
    }

    // Service Worker setup for advanced caching
    async setupServiceWorker() {
        if ('serviceWorker' in navigator) {
            try {
                const registration = await navigator.serviceWorker.register('/static/js/sw.js');
                console.log('Service Worker registered:', registration);
            } catch (error) {
                console.warn('Service Worker registration failed:', error);
            }
        }
    }

    // Graceful degradation helpers
    showOfflineFallback(container) {
        if (!container) return;

        container.innerHTML = `
            <div class="text-center py-12">
                <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 5.636l-3.536 3.536m0 5.656l3.536 3.536M9.172 9.172L5.636 5.636m3.536 9.192L5.636 18.364M12 2.25a9.75 9.75 0 100 19.5 9.75 9.75 0 000-19.5z" />
                </svg>
                <h3 class="mt-2 text-sm font-medium text-gray-900">Offline</h3>
                <p class="mt-1 text-sm text-gray-500">
                    This content is not available offline. Please check your connection and try again.
                </p>
                <div class="mt-6">
                    <button onclick="location.reload()" class="btn-primary">
                        Retry
                    </button>
                </div>
            </div>
        `;
    }

    // Check if specific features are available offline
    isFeatureAvailable(feature) {
        const offlineFeatures = [
            'view-cached-servers',
            'view-cached-tools',
            'basic-navigation'
        ];

        return this.isOnline || offlineFeatures.includes(feature);
    }

    showNotification(message, type = 'info') {
        if (window.notifications) {
            if (type === 'success') {
                window.notifications.showSuccess(message);
            } else {
                window.notifications.showError(message);
            }
        } else {
            console.log(`${type.toUpperCase()}: ${message}`);
        }
    }

    // Get offline status
    getStatus() {
        return {
            isOnline: this.isOnline,
            queuedRequests: this.offlineQueue.length,
            cachedItems: this.offlineData.size
        };
    }
}

// Simple Service Worker for basic caching
const serviceWorkerCode = `
const CACHE_NAME = 'mcp-jungle-v1';
const urlsToCache = [
    '/',
    '/static/css/app.css',
    '/static/js/app.js',
    '/static/js/offline.js',
    '/404.html'
];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then((cache) => cache.addAll(urlsToCache))
    );
});

self.addEventListener('fetch', (event) => {
    event.respondWith(
        caches.match(event.request)
            .then((response) => {
                // Return cached version or fetch from network
                return response || fetch(event.request);
            })
    );
});
`;

// Create service worker file if it doesn't exist
if ('serviceWorker' in navigator) {
    const blob = new Blob([serviceWorkerCode], { type: 'application/javascript' });
    const swUrl = URL.createObjectURL(blob);

    // Store the service worker code for registration
    window.serviceWorkerBlob = blob;
}

// Initialize offline manager
const offlineManager = new OfflineManager();

// Make available globally
window.OfflineManager = OfflineManager;
window.offlineManager = offlineManager;

// Utility functions
window.isOnline = () => offlineManager.isOnline;
window.cacheData = (key, data) => offlineManager.cacheData(key, data);
window.getCachedData = (key) => offlineManager.getCachedData(key);
window.showOfflineFallback = (container) => offlineManager.showOfflineFallback(container);