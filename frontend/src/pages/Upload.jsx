import { useState, useRef } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { api } from '../services/api'
import { supabase } from '../services/supabase'

export default function Upload() {
    const [file, setFile] = useState(null)
    const [isUploading, setIsUploading] = useState(false)
    const [uploadProgress, setUploadProgress] = useState(0)
    const [error, setError] = useState(null)
    const [dragActive, setDragActive] = useState(false)
    const fileInputRef = useRef(null)
    const navigate = useNavigate()

    const handleFile = (e) => {
        const selectedFile = e.target.files?.[0]
        if (selectedFile) {
            if (selectedFile.size > 100 * 1024 * 1024) {
                setError("El archivo supera el límite de 100MB")
                return
            }
            setFile(selectedFile)
            setError(null)
        }
    }

    const handleDrag = (e) => {
        e.preventDefault()
        e.stopPropagation()
        if (e.type === "dragenter" || e.type === "dragover") {
            setDragActive(true)
        } else if (e.type === "dragleave") {
            setDragActive(false)
        }
    }

    const handleDrop = (e) => {
        e.preventDefault()
        e.stopPropagation()
        setDragActive(false)
        if (e.dataTransfer.files?.[0]) {
            const droppedFile = e.dataTransfer.files[0]
            if (droppedFile.size > 100 * 1024 * 1024) {
                setError("El archivo supera el límite de 100MB")
                return
            }
            setFile(droppedFile)
            setError(null)
        }
    }

    const clearFile = () => {
        setFile(null)
        setUploadProgress(0)
        setError(null)
    }

    const startUpload = async () => {
        if (!file) return
        setIsUploading(true)
        setError(null)

        try {
            // 1. Get signed URL from backend
            const { file_id, upload_url } = await api.uploadFile(file.name, file.size, file.type)

            // 2. Upload directly to Supabase Storage using the signed URL
            return new Promise((resolve, reject) => {
                const xhr = new XMLHttpRequest()
                xhr.open('PUT', upload_url)
                xhr.setRequestHeader('Content-Type', file.type)

                xhr.upload.onprogress = (event) => {
                    if (event.lengthComputable) {
                        const percent = Math.round((event.loaded / event.total) * 100)
                        setUploadProgress(percent)
                    }
                }

                xhr.onload = async () => {
                    if (xhr.status === 200 || xhr.status === 201) {
                        try {
                            // 3. Confirm upload to backend to start transcription worker
                            await api.confirmUpload(file_id)
                            navigate(`/transcription/${file_id}`)
                            resolve()
                        } catch (err) {
                            reject(err)
                        }
                    } else {
                        reject(new Error(`Error al subir: ${xhr.statusText}`))
                    }
                }

                xhr.onerror = () => reject(new Error("Error de red durante la carga"))
                xhr.send(file)
            })

        } catch (err) {
            setError(err.message)
            setIsUploading(false)
        }
    }

    return (
        <div className="page">
            <div className="container" style={{ maxWidth: '800px' }}>
                <header style={{ marginBottom: '48px' }}>
                    <Link to="/" style={{ color: 'var(--color-text-secondary)', fontSize: '0.875rem', marginBottom: '16px', display: 'block' }}>
                        ← Volver al Dashboard
                    </Link>
                    <h1>Subir Audio</h1>
                    <p style={{ color: 'var(--color-text-secondary)' }}>Formatos soportados: MP3, WAV, M4A (Máx. 100MB)</p>
                </header>

                <div className="card">
                    {!isUploading ? (
                        <>
                            {!file ? (
                                <div
                                    className={`dropzone ${dragActive ? 'dragover' : ''}`}
                                    onDragEnter={handleDrag}
                                    onDragLeave={handleDrag}
                                    onDragOver={handleDrag}
                                    onDrop={handleDrop}
                                    onClick={() => fileInputRef.current.click()}
                                >
                                    <input
                                        type="file"
                                        ref={fileInputRef}
                                        onChange={handleFile}
                                        style={{ display: 'none' }}
                                        accept="audio/*"
                                    />
                                    <div className="dropzone-icon">☁️</div>
                                    <div>
                                        <h3>Haz clic o arrastra un archivo</h3>
                                        <p style={{ color: 'var(--color-text-muted)', fontSize: '0.875rem' }}>
                                            Arrastra tu entrevista, podcast o nota de voz aquí
                                        </p>
                                    </div>
                                </div>
                            ) : (
                                <div style={{ textAlign: 'center', padding: '24px' }}>
                                    <div style={{ fontSize: '3rem', marginBottom: '16px' }}>📄</div>
                                    <h3 style={{ marginBottom: '8px' }}>{file.name}</h3>
                                    <p style={{ color: 'var(--color-text-muted)', marginBottom: '24px' }}>
                                        {(file.size / 1024 / 1024).toFixed(2)} MB
                                    </p>
                                    <div style={{ display: 'flex', gap: '12px', justifyContent: 'center' }}>
                                        <button className="btn btn-secondary" onClick={clearFile}>
                                            Cambiar Archivo
                                        </button>
                                        <button className="btn btn-primary" onClick={startUpload}>
                                            Comenzar Transcripción
                                        </button>
                                    </div>
                                </div>
                            )}

                            {error && (
                                <div style={{ color: 'var(--color-error)', marginTop: '16px', fontSize: '0.875rem', textAlign: 'center' }}>
                                    {error}
                                </div>
                            )}
                        </>
                    ) : (
                        <div style={{ textAlign: 'center', padding: '48px' }}>
                            <h3 style={{ marginBottom: '24px' }}>Subiendo tu archivo...</h3>
                            <div className="progress-container" style={{ marginBottom: '16px' }}>
                                <div className="progress-bar" style={{ width: `${uploadProgress}%` }}></div>
                            </div>
                            <p className="mono" style={{ color: 'var(--color-text-primary)' }}>{uploadProgress}%</p>
                            <p style={{ color: 'var(--color-text-muted)', marginTop: '24px' }}>
                                No cierres esta pestaña hasta que la carga se complete.
                            </p>
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}
