import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../services/api'
import { wsClient } from '../services/websocket'

export default function Transcription() {
    const { id } = useParams()
    const [data, setData] = useState(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)
    const [wsStatus, setWsStatus] = useState(null)

    useEffect(() => {
        fetchTranscription()

        // Subscribe to real-time updates
        const unsubscribe = wsClient.subscribe(id, (update) => {
            setWsStatus(update)
            if (update.status === 'completed') {
                fetchTranscription()
            }
        })

        return () => unsubscribe()
    }, [id])

    const fetchTranscription = async () => {
        try {
            setLoading(true)
            const result = await api.getTranscription(id)
            setData(result)
            setError(null)
        } catch (err) {
            if (err.message.includes('not ready')) {
                // Expected if still processing
                setWsStatus({ status: 'processing', message: 'Estamos trabajando en ello...' })
            } else {
                setError(err.message)
            }
        } finally {
            setLoading(false)
        }
    }

    const handleExport = async () => {
        try {
            const { content, file_name } = await api.exportTranscription(id)
            const blob = new Blob([content], { type: 'text/plain' })
            const url = window.URL.createObjectURL(blob)
            const a = document.createElement('a')
            a.href = url
            a.download = file_name
            a.click()
            window.URL.revokeObjectURL(url)
        } catch (err) {
            alert("Error al exportar: " + err.message)
        }
    }

    const copyToClipboard = () => {
        if (!data?.transcription?.full_text) return
        navigator.clipboard.writeText(data.transcription.full_text)
        alert("Copiado al portapapeles")
    }

    if (error) {
        return (
            <div className="page flex items-center justify-center">
                <div className="card text-center" style={{ maxWidth: '600px' }}>
                    <div style={{ fontSize: '3rem', marginBottom: '16px' }}>❌</div>
                    <h3>Error al cargar</h3>
                    <p style={{ color: 'var(--color-error)', marginBottom: '24px' }}>{error}</p>
                    <Link to="/" className="btn btn-secondary">Volver al Dashboard</Link>
                </div>
            </div>
        )
    }

    const isProcessing = data?.media_file?.status === 'processing' || wsStatus?.status === 'processing'

    return (
        <div className="page">
            <div className="container">
                <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '40px' }}>
                    <div>
                        <Link to="/" style={{ color: 'var(--color-text-secondary)', fontSize: '0.875rem', marginBottom: '8px', display: 'block' }}>
                            ← Mis Proyectos
                        </Link>
                        <h1>{data?.media_file?.file_name || 'Cargando...'}</h1>
                        <div style={{ display: 'flex', gap: '12px', marginTop: '12px', alignItems: 'center' }}>
                            {data?.transcription?.language && (
                                <span className="badge" style={{ background: 'var(--color-bg-elevated)', color: 'var(--color-text-secondary)' }}>
                                    🌐 {data.transcription.language.toUpperCase()}
                                </span>
                            )}
                            {data?.media_file?.status === 'completed' ? (
                                <span className="badge badge-completed">✓ Completado</span>
                            ) : (
                                <span className="badge badge-processing">⚙ {wsStatus?.message || 'Procesando...'}</span>
                            )}
                        </div>
                    </div>

                    {data?.media_file?.status === 'completed' && (
                        <div style={{ display: 'flex', gap: '12px' }}>
                            <button className="btn btn-secondary" onClick={copyToClipboard}>
                                Copiar Texto
                            </button>
                            <button className="btn btn-primary" onClick={handleExport}>
                                Exportar TXT
                            </button>
                        </div>
                    )}
                </header>

                {isProcessing ? (
                    <div className="card text-center animate-fade-in" style={{ padding: '80px 24px' }}>
                        <div className="waveform" style={{ justifyContent: 'center', marginBottom: '32px', height: '48px' }}>
                            <div className="waveform-bar" style={{ width: '6px' }}></div>
                            <div className="waveform-bar" style={{ width: '6px' }}></div>
                            <div className="waveform-bar" style={{ width: '6px' }}></div>
                            <div className="waveform-bar" style={{ width: '6px' }}></div>
                            <div className="waveform-bar" style={{ width: '6px' }}></div>
                            <div className="waveform-bar" style={{ width: '6px' }}></div>
                        </div>
                        <h3>Tu transcripción está en camino</h3>
                        <p style={{ color: 'var(--color-text-secondary)', maxWidth: '400px', margin: '16px auto 32px' }}>
                            Nuestra IA está escuchando cada palabra. Esto suele tardar un tercio de la duración del audio.
                        </p>
                        <div className="badge badge-processing animate-pulse" style={{ padding: '8px 20px', fontSize: '1rem' }}>
                            {wsStatus?.message || 'Procesando...'}
                        </div>
                    </div>
                ) : data?.transcription ? (
                    <div className="animate-fade-in" style={{ display: 'grid', gridTemplateColumns: '1fr', gap: '32px' }}>
                        <div className="card workplace">
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px', borderBottom: '1px solid var(--color-border)', paddingBottom: '16px' }}>
                                <h3 className="mono" style={{ color: 'var(--color-text-secondary)', fontSize: '0.875rem' }}>TEXTO GENERADO</h3>
                                <span className="mono" style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)' }}>
                                    {data.transcription.word_count} palabras
                                </span>
                            </div>

                            <div style={{ lineHeight: '1.8', fontSize: '1.125rem', whiteSpace: 'pre-wrap' }}>
                                {data.transcription.full_text}
                            </div>

                            <div style={{ marginTop: '48px' }}>
                                <h3 className="mono" style={{ color: 'var(--color-text-secondary)', fontSize: '0.875rem', marginBottom: '24px' }}>SEGMENTOS CON TIEMPO</h3>
                                <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                                    {data.segments.map((seg) => (
                                        <div key={seg.id} style={{ display: 'flex', gap: '24px', padding: '12px', borderRadius: '8px', transition: 'background 0.2s' }} className="segment-row">
                                            <div className="mono" style={{ color: 'var(--color-primary)', fontSize: '0.875rem', minWidth: '60px', opacity: 0.7 }}>
                                                {new Date(seg.start_time * 1000).toISOString().substr(14, 5)}
                                            </div>
                                            <div>{seg.content}</div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </div>
                    </div>
                ) : null}
            </div>

            <style dangerouslySetInnerHTML={{
                __html: `
        .segment-row:hover {
          background: var(--color-bg-elevated);
        }
      `}} />
        </div>
    )
}
