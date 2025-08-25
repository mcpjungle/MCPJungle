// Security utilities for the web UI
class SecurityManager {
    constructor() {
        this.csrfToken = null;
        this.init();
    }

    init() {
        this.setupCSRFProtection();
        this.setupInputSanitization();
        this.setupSecurityEventListeners();
    }

    // CSRF Protection
    setupCSRFProtection() {
        // Generate a simple CSRF token for client-side protection
        this.csrfToken = this.generateCSRFToken();

        // Add CSRF token to all forms
        this.addCSRFTokenToForms();

        // Intercept fetch requests to add CSRF token
        this.interceptFetchRequests();
    }

    generateCSRFToken() {
        const array = new Uint8Array(32);
        crypto.getRandomValues(array);
        return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
    }

    addCSRFTokenToForms() {
        document.querySelectorAll('form').forEach(form => {
            if (!form.querySelector('input[name="csrf_token"]')) {
                const csrfInput = document.createElement('input');
                csrfInput.type = 'hidden';
                csrfInput.name = 'csrf_token';
                csrfInput.value = this.csrfToken;
                form.appendChild(csrfInput);
            }
        });
    }

    interceptFetchRequests() {
        const originalFetch = window.fetch;
        window.fetch = async (url, options = {}) => {
            // Add CSRF token to POST, PUT, DELETE requests
            if (options.method && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(options.method.toUpperCase())) {
                options.headers = {
                    'X-CSRF-Token': this.csrfToken,
                    ...options.headers
                };
            }

            return originalFetch(url, options);
        };
    }

    // Input Sanitization
    setupInputSanitization() {
        // Add input event listeners for real-time sanitization
        document.addEventListener('input', (e) => {
            if (e.target.matches('input[type="text"], textarea')) {
                this.sanitizeInput(e.target);
            }
        });
    }

    sanitizeInput(element) {
        const value = element.value;

        // Basic XSS prevention - remove script tags and javascript: URLs
        const sanitized = value
            .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
            .replace(/javascript:/gi, '')
            .replace(/on\w+\s*=/gi, '');

        if (sanitized !== value) {
            element.value = sanitized;
            this.showSecurityWarning('Potentially unsafe content was removed from your input.');
        }
    }

    // HTML Escaping
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    unescapeHtml(html) {
        const div = document.createElement('div');
        div.innerHTML = html;
        return div.textContent || div.innerText || '';
    }

    // URL Validation
    isValidUrl(string) {
        try {
            const url = new URL(string);
            return ['http:', 'https:'].includes(url.protocol);
        } catch (_) {
            return false;
        }
    }

    // Content Security Policy Violation Handling
    setupSecurityEventListeners() {
        // Listen for CSP violations
        document.addEventListener('securitypolicyviolation', (e) => {
            console.warn('CSP Violation:', {
                blockedURI: e.blockedURI,
                violatedDirective: e.violatedDirective,
                originalPolicy: e.originalPolicy
            });

            // Report to monitoring service if available
            this.reportSecurityViolation(e);
        });

        // Listen for mixed content warnings
        if ('SecurityPolicyViolationEvent' in window) {
            window.addEventListener('securitypolicyviolation', (e) => {
                if (e.violatedDirective.includes('mixed-content')) {
                    this.showSecurityWarning('Mixed content detected. Please ensure all resources are loaded over HTTPS.');
                }
            });
        }
    }

    reportSecurityViolation(violation) {
        // In a real application, you might send this to a monitoring service
        const report = {
            type: 'csp-violation',
            blockedURI: violation.blockedURI,
            violatedDirective: violation.violatedDirective,
            timestamp: new Date().toISOString(),
            userAgent: navigator.userAgent,
            url: window.location.href
        };

        // For now, just log it
        console.warn('Security violation reported:', report);
    }

    // Security Warnings
    showSecurityWarning(message) {
        if (window.notifications) {
            window.notifications.showError(`Security Warning: ${message}`);
        } else {
            console.warn('Security Warning:', message);
        }
    }

    // Secure Random Number Generation
    generateSecureRandom(length = 16) {
        const array = new Uint8Array(length);
        crypto.getRandomValues(array);
        return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
    }

    // Session Security
    setupSessionSecurity() {
        // Clear sensitive data on page unload
        window.addEventListener('beforeunload', () => {
            this.clearSensitiveData();
        });

        // Detect if page is being loaded in an iframe (clickjacking protection)
        if (window.top !== window.self) {
            this.showSecurityWarning('This page should not be loaded in a frame.');
        }

        // Basic session timeout warning
        this.setupSessionTimeout();
    }

    setupSessionTimeout() {
        let lastActivity = Date.now();
        const SESSION_TIMEOUT = 30 * 60 * 1000; // 30 minutes
        const WARNING_TIME = 5 * 60 * 1000; // 5 minutes before timeout

        const updateActivity = () => {
            lastActivity = Date.now();
        };

        // Track user activity
        ['mousedown', 'mousemove', 'keypress', 'scroll', 'touchstart'].forEach(event => {
            document.addEventListener(event, updateActivity, true);
        });

        // Check for session timeout
        setInterval(() => {
            const timeSinceActivity = Date.now() - lastActivity;

            if (timeSinceActivity > SESSION_TIMEOUT) {
                this.handleSessionTimeout();
            } else if (timeSinceActivity > SESSION_TIMEOUT - WARNING_TIME) {
                this.showSessionWarning();
            }
        }, 60000); // Check every minute
    }

    handleSessionTimeout() {
        this.showSecurityWarning('Your session has expired due to inactivity. Please refresh the page.');
        this.clearSensitiveData();
    }

    showSessionWarning() {
        if (window.notifications) {
            window.notifications.showError('Your session will expire soon due to inactivity.');
        }
    }

    clearSensitiveData() {
        // Clear any sensitive data from memory
        this.csrfToken = null;

        // Clear form data
        document.querySelectorAll('input[type="password"]').forEach(input => {
            input.value = '';
        });
    }

    // Secure Form Submission
    secureFormSubmit(form, data) {
        // Add CSRF token
        data.csrf_token = this.csrfToken;

        // Sanitize all string values
        Object.keys(data).forEach(key => {
            if (typeof data[key] === 'string') {
                data[key] = this.sanitizeInput({ value: data[key] }).value;
            }
        });

        return data;
    }

    // Content Validation
    validateContent(content, type = 'text') {
        switch (type) {
            case 'url':
                return this.isValidUrl(content);
            case 'email':
                return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(content);
            case 'text':
                return typeof content === 'string' && content.length > 0;
            default:
                return true;
        }
    }
}

// Initialize security manager
const securityManager = new SecurityManager();

// Make available globally
window.SecurityManager = SecurityManager;
window.securityManager = securityManager;

// Initialize session security when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    securityManager.setupSessionSecurity();
});

// Export utilities for use in other scripts
window.escapeHtml = (text) => securityManager.escapeHtml(text);
window.sanitizeInput = (element) => securityManager.sanitizeInput(element);
window.validateContent = (content, type) => securityManager.validateContent(content, type);