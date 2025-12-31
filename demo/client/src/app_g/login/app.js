// Login page app - minimal initialization

import { AuthManager } from './auth-manager.js';

document.addEventListener('DOMContentLoaded', function () {
    // Focus username field on load
    const usernameInput = document.getElementById('username');
    if (usernameInput) {
        usernameInput.focus();
    }
});


const auth = new AuthManager();

const loginForm = document.getElementById('login-form');
const registerForm = document.getElementById('register-form');
const errorMsg = document.getElementById('error-message');
const successMsg = document.getElementById('success-message');

function showError(msg) {
    errorMsg.textContent = msg;
    errorMsg.style.display = 'block';
    successMsg.style.display = 'none';
}

function showSuccess(msg) {
    successMsg.textContent = msg;
    successMsg.style.display = 'block';
    errorMsg.style.display = 'none';
}

function hideMessages() {
    errorMsg.style.display = 'none';
    successMsg.style.display = 'none';
}

function setLoading(btn, loading) {
    const text = btn.querySelector('.btn-text');
    const loader = btn.querySelector('.btn-loader');
    if (loading) {
        text.style.display = 'none';
        loader.style.display = 'inline';
        btn.disabled = true;
    } else {
        text.style.display = 'inline';
        loader.style.display = 'none';
        btn.disabled = false;
    }
}

// Toggle between login and register forms
document.getElementById('show-register').addEventListener('click', (e) => {
    e.preventDefault();
    loginForm.style.display = 'none';
    registerForm.style.display = 'block';
    hideMessages();
});

document.getElementById('show-login').addEventListener('click', (e) => {
    e.preventDefault();
    registerForm.style.display = 'none';
    loginForm.style.display = 'block';
    hideMessages();
});

// Login form submission
loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    hideMessages();

    const btn = document.getElementById('login-btn');
    const username = document.getElementById('username').value.trim();
    const password = document.getElementById('password').value;

    setLoading(btn, true);

    try {
        const result = await auth.login(username, password);
        if (result.success) {
            showSuccess(`Welcome, ${result.username}! Redirecting...`);
            setTimeout(() => {
                console.log("PATH PREFIX:", window.PATH_PREFIX);
                window.location.href = '/w/' + window.PATH_PREFIX.split('/').pop() + '/home/';
            }, 1000);
        } else {
            showError(result.error || 'Login failed');
        }
    } catch (err) {
        showError(err.message || 'Network error');
    } finally {
        setLoading(btn, false);
    }
});

// Register form submission
registerForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    hideMessages();

    const btn = document.getElementById('register-btn');
    const username = document.getElementById('reg-username').value.trim();
    const password = document.getElementById('reg-password').value;

    setLoading(btn, true);

    try {
        const result = await auth.register(username, password);
        if (result.success) {
            showSuccess('Account created! You can now sign in.');
            registerForm.style.display = 'none';
            loginForm.style.display = 'block';
            document.getElementById('username').value = username;
        } else {
            showError(result.error || 'Registration failed');
        }
    } catch (err) {
        showError(err.message || 'Network error');
    } finally {
        setLoading(btn, false);
    }
});