// Authentication system for MCP Jungle Web UI
class AuthManager {
    constructor() {
        this.token = null;
        this.user = null;
        this.mode = 'dev'; // Default to dev mode
        this.isAuthenticated = false;
        this.init();
    }

    async init() {
        await this.loadStoredAuth();
        await this.checkServerMode();
        this.setupAuthInterceptors();
        this.checkAuthStatus();
    }

    // Load stored authentication from localStorage
    loadStoredAuth() {
        try {
            const storedToken = localStorage.getItem('mcp_jungle_token');
            const storedUser = localStorage.getItem('mcp_jungle_user');

            if (storedToken) {
                this.token = storedToken;
            }

            if (storedUser) {
                this.user = JSON.parse(storedUser);
            }

            this.isAuthenticated = !!(this.token && this.user);
        } catch (error) {
            console.warn('Failed to load stored auth:', error);
            this.clearAuth();
        }
    }

    // Check server mode (dev/prod) to determine auth requirements
    async checkServerMode() {
        try {
            // Try to make a request to determine server mode
            const response = await fetch('/health');
            if (response.ok) {
                // In dev mode, we don't need authentication for most operations
                // We'll detect prod mode when we get 401 responses
                this.mode = 'dev';
            }
        } catch (error) {
            console.warn('Failed to check server mode:', error);
        }
    }

    // Setup request interceptors to add authentication headers
    setupAuthInterceptors() {
        const originalFetch = window.fetch;

        window.fetch = async (url, options = {}) => {
            // Add authentication header if we have a token and it's an API request
            if (this.token && url.includes('/api/')) {
                options.headers = {
                    'Authorization': `Bearer ${this.token}`,
                    ...options.headers
                };
            }

            try {
                const response = await originalFetch(url, options);

                // Handle authentication errors
                if (response.status === 401) {
                    this.handleAuthError();
                }

                // Detect production mode from 401 responses
                if (response.status === 401 && url.includes('/api/')) {
                    this.mode = 'prod';
                }

                return response;
            } catch (error) {
                throw error;
            }
        };
    }

    // Check current authentication status
    async checkAuthStatus() {
        if (!this.token) {
            this.isAuthenticated = false;
            return;
        }

        try {
            // Try to make an authenticated request to verify token
            const response = await fetch('/api/v0/servers');

            if (response.ok) {
                this.isAuthenticated = true;
            } else if (response.status === 401) {
                this.handleAuthError();
            }
        } catch (error) {
            console.warn('Failed to check auth status:', error);
            this.isAuthenticated = false;
        }
    }

    // Handle authentication errors
    handleAuthError() {
        this.clearAuth();
        this.showAuthRequired();
    }

    // Show authentication required message/modal
    showAuthRequired() {
        if (this.mode === 'dev') {
            // In dev mode, authentication errors might indicate server issues
            if (window.notifications) {
                window.notifications.showError('Server authentication error. Please check server configuration.');
            }
            return;
        }

        // In production mode, show login form
        this.showLoginModal();
    }

    // Show login modal for production mode
    showLoginModal() {
        // Create login modal if it doesn't exist
        let modal = document.getElementById('auth-modal');
        if (!modal) {
            modal = this.createLoginModal();
            document.body.appendChild(modal);
        }

        // Show modal using Alpine.js if available
        if (window.Alpine) {
            Alpine.store('auth', { showLoginModal: true });
        } else {
            modal.style.display = 'block';
        }
    }

    // Create login modal HTML
    createLoginModal() {
        const modal = document.createElement('div');
        modal.id = 'auth-modal';
        modal.className = 'fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50';
        modal.innerHTML = `
            <div class="relative top-20 mx-auto p-5 border w-96 shadow-lg rounded-md bg-white">
                <div class="mt-3">
                    <h3 class="text-lg font-medium text-gray-900 mb-4">Authentication Required</h3>
                    <p class="text-sm text-gray-600 mb-4">
                        This server is running in production mode and requires authentication.
                    </p>
                    <form id="login-form">
                        <div class="mb-4">
                            <label class="form-label">Access Token</label>
                            <input type="password" id="auth-token" class="form-input"
                                   placeholder="Enter your admin access token" required>
                            <p class="mt-1 text-sm text-gray-500">
                                Contact your administrator for an access token.
                            </p>
                        </div>
                        <div class="flex justify-end space-x-3">
                            <button type="button" onclick="authManager.hideLoginModal()" class="btn-secondary">
                                Cancel
                            </button>
                            <button type="submit" class="btn-primary">
                                Authenticate
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        `;

        // Add form submit handler
        modal.querySelector('#login-form').addEventListener('submit', (e) => {
            e.preventDefault();
            const token = modal.querySelector('#auth-token').value;
            this.authenticate(token);
        });

        return modal;
    }

    // Hide login modal
    hideLoginModal() {
        const modal = document.getElementById('auth-modal');
        if (modal) {
            modal.style.display = 'none';
        }

        if (window.Alpine) {
            Alpine.store('auth', { showLoginModal: false });
        }
    }

    // Authenticate with token
    async authenticate(token) {
        if (!token) {
            this.showError('Please enter an access token');
            return;
        }

        try {
            // Test the token by making an authenticated request
            const response = await fetch('/api/v0/servers', {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            if (response.ok) {
                this.token = token;
                this.user = { token }; // Minimal user object
                this.isAuthenticated = true;

                // Store authentication
                localStorage.setItem('mcp_jungle_token', token);
                localStorage.setItem('mcp_jungle_user', JSON.stringify(this.user));

                this.hideLoginModal();
                this.showSuccess('Authentication successful');

                // Reload the page to refresh data
                window.location.reload();
            } else {
                this.showError('Invalid access token');
            }
        } catch (error) {
            console.error('Authentication failed:', error);
            this.showError('Authentication failed. Please try again.');
        }
    }

    // Logout
    logout() {
        this.clearAuth();
        this.showSuccess('Logged out successfully');

        // Redirect to home page
        window.location.href = '/';
    }

    // Clear authentication data
    clearAuth() {
        this.token = null;
        this.user = null;
        this.isAuthenticated = false;

        // Clear stored data
        localStorage.removeItem('mcp_jungle_token');
        localStorage.removeItem('mcp_jungle_user');
    }

    // Check if user is authenticated
    isAuth() {
        return this.isAuthenticated;
    }

    // Get current token
    getToken() {
        return this.token;
    }

    // Get current user
    getUser() {
        return this.user;
    }

    // Get server mode
    getMode() {
        return this.mode;
    }

    // Check if authentication is required
    isAuthRequired() {
        return this.mode === 'prod';
    }

    // Show success message
    showSuccess(message) {
        if (window.notifications) {
            window.notifications.showSuccess(message);
        } else {
            console.log('SUCCESS:', message);
        }
    }

    // Show error message
    showError(message) {
        if (window.notifications) {
            window.notifications.showError(message);
        } else {
            console.error('ERROR:', message);
        }
    }

    // Get authentication status for UI
    getAuthStatus() {
        return {
            isAuthenticated: this.isAuthenticated,
            mode: this.mode,
            user: this.user,
            requiresAuth: this.isAuthRequired()
        };
    }
}

// Authentication guard for protecting routes/features
class AuthGuard {
    constructor(authManager) {
        this.auth = authManager;
    }

    // Check if user can access a feature
    canAccess(feature = 'default') {
        // In dev mode, allow all access
        if (this.auth.getMode() === 'dev') {
            return true;
        }

        // In prod mode, require authentication
        return this.auth.isAuth();
    }

    // Require authentication for a function
    requireAuth(fn, fallback = null) {
        return (...args) => {
            if (this.canAccess()) {
                return fn(...args);
            } else {
                if (fallback) {
                    return fallback(...args);
                } else {
                    this.auth.showAuthRequired();
                    return null;
                }
            }
        };
    }

    // Protect an element (hide/show based on auth)
    protectElement(element, showWhenAuth = true) {
        const updateVisibility = () => {
            const canAccess = this.canAccess();
            const shouldShow = showWhenAuth ? canAccess : !canAccess;

            if (element) {
                element.style.display = shouldShow ? '' : 'none';
            }
        };

        // Initial update
        updateVisibility();

        // Update when auth status changes
        // This would need to be integrated with auth state changes
        return updateVisibility;
    }
}

// Initialize authentication
const authManager = new AuthManager();
const authGuard = new AuthGuard(authManager);

// Make available globally
window.AuthManager = AuthManager;
window.AuthGuard = AuthGuard;
window.authManager = authManager;
window.authGuard = authGuard;

// Utility functions
window.isAuthenticated = () => authManager.isAuth();
window.requireAuth = (fn, fallback) => authGuard.requireAuth(fn, fallback);
window.getAuthToken = () => authManager.getToken();

// Alpine.js store for auth state (if Alpine is available)
document.addEventListener('alpine:init', () => {
    if (window.Alpine) {
        Alpine.store('auth', {
            showLoginModal: false,
            isAuthenticated: authManager.isAuth(),
            mode: authManager.getMode(),
            user: authManager.getUser(),

            login(token) {
                authManager.authenticate(token);
            },

            logout() {
                authManager.logout();
            },

            getStatus() {
                return authManager.getAuthStatus();
            }
        });
    }
});