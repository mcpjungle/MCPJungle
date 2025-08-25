// Main application JavaScript for MCP Jungle Web UI

// Navigation component
function navigationComponent() {
    return {
        currentPath: window.location.pathname,

        init() {
            // Update current path when navigating
            this.updateCurrentPath();
        },

        updateCurrentPath() {
            this.currentPath = window.location.pathname;
        },

        isActive(path) {
            return this.currentPath === path;
        },

        navigate(path) {
            window.location.href = path;
        }
    }
}

// Global error handling and notifications
function errorHandler() {
    return {
        errors: [],
        successes: [],

        showError(message, type = 'error') {
            const error = {
                id: Date.now() + Math.random(),
                message,
                type,
                timestamp: new Date()
            };
            this.errors.push(error);

            // Auto-remove after 5 seconds
            setTimeout(() => {
                this.removeError(error.id);
            }, 5000);
        },

        showSuccess(message) {
            const success = {
                id: Date.now() + Math.random(),
                message,
                timestamp: new Date()
            };
            this.successes.push(success);

            // Auto-remove after 3 seconds
            setTimeout(() => {
                this.removeSuccess(success.id);
            }, 3000);
        },

        removeError(id) {
            this.errors = this.errors.filter(error => error.id !== id);
        },

        removeSuccess(id) {
            this.successes = this.successes.filter(success => success.id !== id);
        },

        clearErrors() {
            this.errors = [];
        },

        clearSuccesses() {
            this.successes = [];
        },

        clearAll() {
            this.clearErrors();
            this.clearSuccesses();
        }
    }
}

// Loading state management
function loadingManager() {
    return {
        loading: {},

        setLoading(key, state = true) {
            this.loading[key] = state;
        },

        isLoading(key) {
            return this.loading[key] || false;
        },

        clearLoading(key) {
            delete this.loading[key];
        }
    }
}

// API utility functions
const API = {
    baseURL: '/api/v0',

    async request(endpoint, options = {}) {
        const url = `${this.baseURL}${endpoint}`;
        const config = {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        };

        try {
            const response = await fetch(url, config);

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`);
            }

            // Handle empty responses
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                return await response.json();
            }

            return null;
        } catch (error) {
            console.error('API request failed:', error);
            throw error;
        }
    },

    async get(endpoint) {
        return this.request(endpoint);
    },

    async post(endpoint, data) {
        return this.request(endpoint, {
            method: 'POST',
            body: JSON.stringify(data)
        });
    },

    async delete(endpoint) {
        return this.request(endpoint, {
            method: 'DELETE'
        });
    }
};

// Make components globally available
window.navigationComponent = navigationComponent;
window.errorHandler = errorHandler;
window.loadingManager = loadingManager;
window.API = API;

// Make individual functions globally available for Alpine.js
window.isActive = function (path) {
    return window.location.pathname === path;
};

// Global notification functions
window.showError = function (message) {
    console.error(message);
};

window.showSuccess = function (message) {
    console.log(message);
};
// Form validation utilities
const FormValidator = {
    rules: {
        required: (value) => value && value.trim() !== '',
        email: (value) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value),
        url: (value) => {
            try {
                new URL(value);
                return true;
            } catch {
                return false;
            }
        },
        minLength: (min) => (value) => value && value.length >= min,
        maxLength: (max) => (value) => value && value.length <= max
    },

    validate(value, rules) {
        const errors = [];

        for (const rule of rules) {
            if (typeof rule === 'string') {
                if (!this.rules[rule](value)) {
                    errors.push(this.getErrorMessage(rule, value));
                }
            } else if (typeof rule === 'object') {
                const [ruleName, param] = Object.entries(rule)[0];
                if (!this.rules[ruleName](param)(value)) {
                    errors.push(this.getErrorMessage(ruleName, value, param));
                }
            } else if (typeof rule === 'function') {
                const result = rule(value);
                if (result !== true) {
                    errors.push(result || 'Invalid value');
                }
            }
        }

        return errors;
    },

    getErrorMessage(rule, param) {
        const messages = {
            required: 'This field is required',
            email: 'Please enter a valid email address',
            url: 'Please enter a valid URL',
            minLength: `Must be at least ${param} characters long`,
            maxLength: `Must be no more than ${param} characters long`
        };

        return messages[rule] || 'Invalid value';
    }
};

// Global notification system
function createGlobalNotifications() {
    const notifications = errorHandler();

    // Make it globally accessible
    window.showError = (message) => notifications.showError(message);
    window.showSuccess = (message) => notifications.showSuccess(message);

    return notifications;
}

// Enhanced API with better error handling
const EnhancedAPI = {
    ...API,

    async request(endpoint, options = {}) {
        try {
            return await API.request(endpoint, options);
        } catch (error) {
            // Show global error notification
            if (window.showError) {
                window.showError(error.message);
            }
            throw error;
        }
    },

    // Retry mechanism for failed requests
    async requestWithRetry(endpoint, options = {}, maxRetries = 3) {
        let lastError;

        for (let i = 0; i <= maxRetries; i++) {
            try {
                return await this.request(endpoint, options);
            } catch (error) {
                lastError = error;

                // Don't retry on client errors (4xx)
                if (error.message.includes('HTTP 4')) {
                    throw error;
                }

                // Wait before retrying (exponential backoff)
                if (i < maxRetries) {
                    await new Promise(resolve => setTimeout(resolve, Math.pow(2, i) * 1000));
                }
            }
        }

        throw lastError;
    }
};

// Connection status monitor
function connectionMonitor() {
    return {
        status: 'connected',
        lastCheck: null,
        checkInterval: null,

        init() {
            this.startMonitoring();

            // Listen for online/offline events
            window.addEventListener('online', () => {
                this.status = 'connected';
                this.checkConnection();
            });

            window.addEventListener('offline', () => {
                this.status = 'disconnected';
            });
        },

        startMonitoring() {
            // Check connection every 30 seconds
            this.checkInterval = setInterval(() => {
                this.checkConnection();
            }, 30000);

            // Initial check
            this.checkConnection();
        },

        async checkConnection() {
            try {
                await fetch('/health', {
                    method: 'GET',
                    cache: 'no-cache',
                    signal: AbortSignal.timeout(5000) // 5 second timeout
                });
                this.status = 'connected';
                this.lastCheck = new Date();
            } catch (error) {
                this.status = 'disconnected';
                console.warn('Connection check failed:', error);
            }
        },

        isConnected() {
            return this.status === 'connected';
        },

        destroy() {
            if (this.checkInterval) {
                clearInterval(this.checkInterval);
            }
        }
    }
}

// Make enhanced utilities globally available
window.FormValidator = FormValidator;
window.EnhancedAPI = EnhancedAPI;
window.connectionMonitor = connectionMonitor;
window.createGlobalNotifications = createGlobalNotifications;

// Keyboard navigation utilities
const KeyboardNav = {
    init() {
        this.setupGlobalKeyboardHandlers();
        this.setupFocusManagement();
    },

    setupGlobalKeyboardHandlers() {
        document.addEventListener('keydown', (e) => {
            // ESC key to close modals
            if (e.key === 'Escape') {
                this.closeTopModal();
            }

            // Tab navigation improvements
            if (e.key === 'Tab') {
                this.handleTabNavigation(e);
            }

            // Arrow key navigation for lists
            if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
                this.handleArrowNavigation(e);
            }
        });
    },

    setupFocusManagement() {
        // Add keyboard navigation class to body
        document.body.classList.add('keyboard-nav');

        // Track focus for better UX
        document.addEventListener('focusin', (e) => {
            this.currentFocus = e.target;
        });
    },

    closeTopModal() {
        // Find and close the topmost modal with confirmation if needed
        const modals = document.querySelectorAll('[x-show*="Modal"]');
        modals.forEach(modal => {
            const alpineData = Alpine.$data(modal);
            if (alpineData) {
                // Check for open modals and handle them appropriately
                Object.keys(alpineData).forEach(key => {
                    if (key.includes('Modal') && alpineData[key] === true) {
                        // Check if this is a form modal with unsaved changes
                        if (this.hasUnsavedChanges(alpineData, key)) {
                            this.confirmModalClose(alpineData, key);
                        } else {
                            alpineData[key] = false;
                        }
                    }
                });
            }
        });
    },

    hasUnsavedChanges(alpineData, modalKey) {
        // Check if there are unsaved changes in form modals
        if (modalKey.includes('Add') || modalKey.includes('Edit')) {
            const form = alpineData.serverForm || alpineData.form || {};

            // Check if any form fields have content
            const hasContent = Object.values(form).some(value =>
                value && typeof value === 'string' && value.trim() !== ''
            );

            return hasContent;
        }
        return false;
    },

    confirmModalClose(alpineData, modalKey) {
        const confirmed = confirm(
            'You have unsaved changes. Are you sure you want to close this dialog? Your changes will be lost.'
        );

        if (confirmed) {
            alpineData[modalKey] = false;
            // Clear the form if it exists
            if (alpineData.closeModal && typeof alpineData.closeModal === 'function') {
                alpineData.closeModal();
            }
        }
    },

    handleTabNavigation(e) {
        // Trap focus within modals
        const modal = e.target.closest('.modal-content');
        if (modal) {
            const focusableElements = modal.querySelectorAll(
                'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
            );
            const firstElement = focusableElements[0];
            const lastElement = focusableElements[focusableElements.length - 1];

            if (e.shiftKey && e.target === firstElement) {
                e.preventDefault();
                lastElement.focus();
            } else if (!e.shiftKey && e.target === lastElement) {
                e.preventDefault();
                firstElement.focus();
            }
        }
    },

    handleArrowNavigation(e) {
        // Handle arrow navigation in lists
        const listItem = e.target.closest('[role="listitem"], li, tr');
        if (listItem) {
            const list = listItem.closest('[role="list"], ul, ol, tbody');
            if (list) {
                const items = Array.from(list.children);
                const currentIndex = items.indexOf(listItem);

                let nextIndex;
                if (e.key === 'ArrowDown') {
                    nextIndex = (currentIndex + 1) % items.length;
                } else {
                    nextIndex = (currentIndex - 1 + items.length) % items.length;
                }

                const nextItem = items[nextIndex];
                const focusableElement = nextItem.querySelector('button, a, input, select, textarea') || nextItem;
                if (focusableElement && focusableElement.focus) {
                    e.preventDefault();
                    focusableElement.focus();
                }
            }
        }
    },

    // Announce changes to screen readers
    announce(message, priority = 'polite') {
        const announcer = document.getElementById('screen-reader-announcer') || this.createAnnouncer();
        announcer.setAttribute('aria-live', priority);
        announcer.textContent = message;

        // Clear after announcement
        setTimeout(() => {
            announcer.textContent = '';
        }, 1000);
    },

    createAnnouncer() {
        const announcer = document.createElement('div');
        announcer.id = 'screen-reader-announcer';
        announcer.className = 'sr-only';
        announcer.setAttribute('aria-live', 'polite');
        announcer.setAttribute('aria-atomic', 'true');
        document.body.appendChild(announcer);
        return announcer;
    }
};

// Accessibility utilities
const A11y = {
    // Set focus to element with optional delay
    setFocus(element, delay = 0) {
        setTimeout(() => {
            if (element && element.focus) {
                element.focus();
            }
        }, delay);
    },

    // Improve form accessibility
    enhanceForm(form) {
        const inputs = form.querySelectorAll('input, select, textarea');
        inputs.forEach(input => {
            const label = form.querySelector(`label[for="${input.id}"]`) ||
                input.closest('.form-group')?.querySelector('label');

            if (label && !input.getAttribute('aria-labelledby')) {
                if (!label.id) {
                    label.id = `label-${Math.random().toString(36).substring(2, 11)}`;
                }
                input.setAttribute('aria-labelledby', label.id);
            }

            // Add required indicator
            if (input.required && label) {
                const requiredIndicator = label.querySelector('.required-indicator') ||
                    document.createElement('span');
                requiredIndicator.className = 'required-indicator text-red-500 ml-1';
                requiredIndicator.textContent = '*';
                requiredIndicator.setAttribute('aria-label', 'required');
                if (!label.querySelector('.required-indicator')) {
                    label.appendChild(requiredIndicator);
                }
            }
        });
    }
};

// Initialize accessibility features when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    KeyboardNav.init();

    // Enhance all forms
    document.querySelectorAll('form').forEach(form => {
        A11y.enhanceForm(form);
    });
});

// Make utilities globally available
window.KeyboardNav = KeyboardNav;
window.A11y = A11y;