// Auth class handles authentication with the backend
class Auth {
    constructor() {
        this.token = localStorage.getItem('token');
        this.user = JSON.parse(localStorage.getItem('user') || 'null');
    }

    async register(username, email, password) {
        const response = await fetch('/v1/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, email, password })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Registration failed');
        }

        const data = await response.json();
        this.setAuth(data.token, data.user);
        return data;
    }

    async login(email, password) {
        const response = await fetch('/v1/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Login failed');
        }

        const data = await response.json();
        this.setAuth(data.token, data.user);
        return data;
    }

    async getCurrentUser() {
        const response = await fetch('/v1/auth/me', {
            headers: { 'Authorization': `Bearer ${this.token}` }
        });

        if (!response.ok) {
            this.clearAuth();
            throw new Error('Not authenticated');
        }

        return await response.json();
    }

    setAuth(token, user) {
        this.token = token;
        this.user = user;
        localStorage.setItem('token', token);
        localStorage.setItem('user', JSON.stringify(user));
    }

    clearAuth() {
        this.token = null;
        this.user = null;
        localStorage.removeItem('token');
        localStorage.removeItem('user');
    }

    isAuthenticated() {
        return !!this.token;
    }

    getToken() {
        return this.token;
    }

    getUser() {
        return this.user;
    }
}
