export class WebSocketClient {
    constructor(url, onMessage) {
        this.url = url;
        this.onMessage = onMessage;
        this.ws = null;
        this.reconnectInterval = 3000;
    }

    connect() {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
            console.log('Connected to WS');
        };

        this.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.onMessage(data);
            } catch (e) {
                console.error('WS Parse error', e);
            }
        };

        this.ws.onclose = () => {
            console.log('WS Disconnected, reconnecting...');
            if (this.reconnectInterval) {
                this.reconnectTimer = setTimeout(() => this.connect(), this.reconnectInterval);
            }
        };
    }

    close() {
        this.reconnectInterval = 0; // Prevent reconnect
        if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
        if (this.ws) {
            this.ws.onclose = null; // Prevent reconnect trigger
            this.ws.close();
        }
    }
}
