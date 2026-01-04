// Chat application logic
class ChatApp {
    constructor(auth) {
        this.auth = auth;
        this.currentRoom = null;
        this.wsClient = null;
        this.rooms = [];
    }

    async loadRooms() {
        const response = await fetch('/v1/rooms', {
            headers: { 'Authorization': `Bearer ${this.auth.getToken()}` }
        });
        this.rooms = await response.json();
        this.renderRooms();
    }

    renderRooms() {
        const roomList = document.getElementById('room-list');
        roomList.innerHTML = this.rooms.map(room => `
            <div class="room-item" data-room-id="${room.id}">
                # ${room.name}
            </div>
        `).join('');

        roomList.querySelectorAll('.room-item').forEach(item => {
            item.addEventListener('click', () => {
                const roomID = item.dataset.roomId;
                this.joinRoom(parseInt(roomID));
            });
        });
    }

    async createRoom(name, description) {
        const response = await fetch('/v1/rooms', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.auth.getToken()}`
            },
            body: JSON.stringify({ name, description })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error);
        }

        await this.loadRooms();
    }

    async joinRoom(roomID) {
        const room = this.rooms.find(r => r.id === roomID);
        if (!room) return;

        await fetch(`/v1/rooms/${roomID}/join`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${this.auth.getToken()}` }
        });

        if (this.wsClient) {
            this.wsClient.disconnect();
        }

        this.currentRoom = room;
        document.getElementById('current-room-name').textContent = `# ${room.name}`;
        document.getElementById('message-field').disabled = false;

        document.querySelectorAll('.room-item').forEach(item => {
            item.classList.toggle('active', item.dataset.roomId === roomID.toString());
        });

        await this.loadMessages(roomID);
        this.connectWebSocket(roomID);
    }

    async loadMessages(roomID) {
        const response = await fetch(`/v1/rooms/${roomID}/messages`, {
            headers: { 'Authorization': `Bearer ${this.auth.getToken()}` }
        });
        const messages = await response.json();

        const messagesDiv = document.getElementById('messages');
        messagesDiv.innerHTML = '';
        messages.forEach(msg => this.displayMessage(msg));
    }

    connectWebSocket(roomID) {
        this.wsClient = new WebSocketClient(roomID, this.auth.getToken());
        this.wsClient.onMessage = (msg) => this.displayMessage(msg);
        this.wsClient.connect();
    }

    sendMessage(content) {
        if (this.wsClient && content.trim()) {
            this.wsClient.send(content);
        }
    }

    displayMessage(msg) {
        const messagesDiv = document.getElementById('messages');
        const messageEl = document.createElement('div');

        if (msg.type === 'join' || msg.type === 'leave') {
            messageEl.className = 'message system';
            messageEl.textContent = `* ${msg.content}`;
        } else {
            messageEl.className = 'message';
            const time = msg.created_at ? new Date(msg.created_at).toLocaleTimeString() : new Date().toLocaleTimeString();
            messageEl.innerHTML = `
                <span class="timestamp">[${time}]</span>
                <span class="username">${msg.username}:</span>
                <span class="content">${msg.content}</span>
            `;
        }

        messagesDiv.appendChild(messageEl);
        messagesDiv.scrollTop = messagesDiv.scrollHeight;
    }
}
