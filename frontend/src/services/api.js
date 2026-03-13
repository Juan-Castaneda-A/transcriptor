const API_BASE = '/api'

/**
 * Make an authenticated API call to the Go backend.
 */
async function apiFetch(path, options = {}) {
    const token = localStorage.getItem('access_token')

    const res = await fetch(`${API_BASE}${path}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
            ...options.headers,
        },
    })

    if (!res.ok) {
        const error = await res.json().catch(() => ({ error: 'Unknown error' }))
        throw new Error(error.message || error.error || `API error: ${res.status}`)
    }

    return res.json()
}

export const api = {
    /** Health check */
    health: () => apiFetch('/health'),

    /** Get user's projects for dashboard */
    getProjects: () => apiFetch('/projects'),

    /** Initiate file upload — returns signed URL */
    uploadFile: (fileName, fileSize, mimeType) =>
        apiFetch('/upload', {
            method: 'POST',
            body: JSON.stringify({
                file_name: fileName,
                file_size: fileSize,
                mime_type: mimeType,
            }),
        }),

    /** Confirm upload after file is sent to Supabase Storage */
    confirmUpload: (fileId) =>
        apiFetch('/upload/confirm', {
            method: 'POST',
            body: JSON.stringify({ file_id: fileId }),
        }),

    /** Get transcription result */
    getTranscription: (fileId) => apiFetch(`/transcriptions/${fileId}`),

    /** Export transcription as TXT */
    exportTranscription: (fileId, format = 'txt') =>
        apiFetch(`/transcriptions/${fileId}/export?format=${format}`),
}
