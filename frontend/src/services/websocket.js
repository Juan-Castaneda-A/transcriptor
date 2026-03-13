/**
 * WebSocket client for real-time transcription status updates.
 */
class WebSocketClient {
    constructor() {
        this.ws = null
        this.listeners = new Map()
        this.reconnectAttempts = 0
        this.maxReconnectAttempts = 5
    }

    connect(userId) {
        if (this.ws?.readyState === WebSocket.OPEN) return

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const host = window.location.host
        const url = `${protocol}//${host}/ws?user_id=${userId}`

        this.ws = new WebSocket(url)

        this.ws.onopen = () => {
            console.log('🔌 WebSocket connected')
            this.reconnectAttempts = 0
        }

        this.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data)
                const { file_id, status, message } = data

                // Notify all listeners for this file
                const fileListeners = this.listeners.get(file_id) || []
                fileListeners.forEach((cb) => cb({ status, message }))

                // Notify global listeners
                const globalListeners = this.listeners.get('*') || []
                globalListeners.forEach((cb) => cb({ file_id, status, message }))
            } catch (err) {
                console.error('Failed to parse WS message:', err)
            }
        }

        this.ws.onclose = () => {
            console.log('🔌 WebSocket disconnected')
            if (this.reconnectAttempts < this.maxReconnectAttempts) {
                this.reconnectAttempts++
                const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000)
                console.log(`Reconnecting in ${delay / 1000}s...`)
                setTimeout(() => this.connect(userId), delay)
            }
        }

        this.ws.onerror = (err) => {
            console.error('WebSocket error:', err)
        }
    }

    /**
     * Subscribe to status updates for a specific file or all files ('*').
     * Returns an unsubscribe function.
     */
    subscribe(fileId, callback) {
        if (!this.listeners.has(fileId)) {
            this.listeners.set(fileId, [])
        }
        this.listeners.get(fileId).push(callback)

        return () => {
            const cbs = this.listeners.get(fileId) || []
            this.listeners.set(
                fileId,
                cbs.filter((cb) => cb !== callback)
            )
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close()
            this.ws = null
        }
        this.listeners.clear()
    }
}

export const wsClient = new WebSocketClient()
