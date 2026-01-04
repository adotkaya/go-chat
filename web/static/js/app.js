// Main application initialization
let auth, chatApp;

document.addEventListener('DOMContentLoaded', () => {
    auth = new Auth();

    if (!auth.isAuthenticated()) {
        showAuthModal();
    } else {
        initChatApp();
    }
});

function showAuthModal() {
    document.getElementById('auth-modal').style.display = 'flex';
    document.getElementById('chat-app').style.display = 'none';

    document.getElementById('login-btn').addEventListener('click', handleLogin);
    document.getElementById('register-btn').addEventListener('click', handleRegister);
    document.getElementById('show-register').addEventListener('click', (e) => {
        e.preventDefault();
        document.getElementById('login-form').style.display = 'none';
        document.getElementById('register-form').style.display = 'block';
    });
    document.getElementById('show-login').addEventListener('click', (e) => {
        e.preventDefault();
        document.getElementById('register-form').style.display = 'none';
        document.getElementById('login-form').style.display = 'block';
    });
}

async function handleLogin() {
    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;
    const errorDiv = document.getElementById('login-error');

    try {
        await auth.login(email, password);
        initChatApp();
    } catch (error) {
        errorDiv.textContent = error.message;
    }
}

async function handleRegister() {
    const username = document.getElementById('register-username').value;
    const email = document.getElementById('register-email').value;
    const password = document.getElementById('register-password').value;
    const errorDiv = document.getElementById('register-error');

    try {
        await auth.register(username, email, password);
        initChatApp();
    } catch (error) {
        errorDiv.textContent = error.message;
    }
}

async function initChatApp() {
    document.getElementById('auth-modal').style.display = 'none';
    document.getElementById('chat-app').style.display = 'block';

    chatApp = new ChatApp(auth);
    await chatApp.loadRooms();

    document.getElementById('logout-btn').addEventListener('click', () => {
        auth.clearAuth();
        location.reload();
    });

    document.getElementById('create-room-btn').addEventListener('click', () => {
        document.getElementById('create-room-modal').style.display = 'flex';
    });

    document.getElementById('create-room-submit').addEventListener('click', async () => {
        const name = document.getElementById('room-name').value;
        const description = document.getElementById('room-description').value;
        const errorDiv = document.getElementById('create-room-error');

        try {
            await chatApp.createRoom(name, description);
            document.getElementById('create-room-modal').style.display = 'none';
            document.getElementById('room-name').value = '';
            document.getElementById('room-description').value = '';
            errorDiv.textContent = '';
        } catch (error) {
            errorDiv.textContent = error.message;
        }
    });

    document.getElementById('create-room-cancel').addEventListener('click', () => {
        document.getElementById('create-room-modal').style.display = 'none';
    });

    document.getElementById('message-field').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            const content = e.target.value;
            chatApp.sendMessage(content);
            e.target.value = '';
        }
    });
}
