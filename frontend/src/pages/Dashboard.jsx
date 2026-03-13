import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../services/api'
import { supabase } from '../services/supabase'

export default function Dashboard() {
    const [projects, setProjects] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)

    useEffect(() => {
        fetchProjects()
    }, [])

    const fetchProjects = async () => {
        try {
            setLoading(true)
            const data = await api.getProjects()
            setProjects(data.files || [])
        } catch (err) {
            setError(err.message)
        } finally {
            setLoading(false)
        }
    }

    const handleLogout = () => {
        supabase.auth.signOut()
    }

    const getStatusBadge = (status) => {
        switch (status) {
            case 'pending': return <span className="badge badge-pending">Pendiente</span>
            case 'processing': return <span className="badge badge-processing animate-pulse">Procesando</span>
            case 'completed': return <span className="badge badge-completed">Completado</span>
            case 'error': return <span className="badge badge-error">Error</span>
            default: return <span className="badge">{status}</span>
        }
    }

    return (
        <div className="page">
            <div className="container">
                <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '48px' }}>
                    <div>
                        <h1>Mis Proyectos</h1>
                        <p style={{ color: 'var(--color-text-secondary)' }}>Gestiona tus transcripciones</p>
                    </div>
                    <div style={{ display: 'flex', gap: '12px' }}>
                        <Link to="/upload" className="btn btn-primary">
                            <span>+</span> Nuevo Archivo
                        </Link>
                        <button onClick={handleLogout} className="btn btn-secondary">
                            Cerrar Sesión
                        </button>
                    </div>
                </header>

                {error && (
                    <div className="card" style={{ color: 'var(--color-error)', border: '1px solid var(--color-error)', marginBottom: '24px' }}>
                        Error al cargar proyectos: {error}
                    </div>
                )}

                {loading ? (
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '24px' }}>
                        {[1, 2, 3].map(i => (
                            <div key={i} className="card animate-pulse" style={{ height: '160px' }}></div>
                        ))}
                    </div>
                ) : projects.length === 0 ? (
                    <div className="card text-center" style={{ padding: '80px 24px' }}>
                        <div style={{ fontSize: '3rem', marginBottom: '16px' }}>📁</div>
                        <h3>No tienes proyectos aún</h3>
                        <p style={{ marginBottom: '24px', color: 'var(--color-text-secondary)' }}>
                            Sube tu primer archivo de audio para empezar a transcribir.
                        </p>
                        <Link to="/upload" className="btn btn-primary">
                            Subir mi primer audio
                        </Link>
                    </div>
                ) : (
                    <div className="stagger" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '24px' }}>
                        {projects.map((project) => (
                            <Link key={project.id} to={`/transcription/${project.id}`} style={{ display: 'block' }}>
                                <div className="card">
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '16px' }}>
                                        <div style={{
                                            width: '40px',
                                            height: '40px',
                                            background: 'var(--color-bg-elevated)',
                                            borderRadius: 'var(--radius-md)',
                                            display: 'flex',
                                            alignItems: 'center',
                                            justifyContent: 'center',
                                            fontSize: '1.2rem'
                                        }}>
                                            🎧
                                        </div>
                                        {getStatusBadge(project.status)}
                                    </div>
                                    <h3 style={{ fontSize: '1.125rem', marginBottom: '4px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                        {project.file_name}
                                    </h3>
                                    <div style={{ fontSize: '0.8125rem', color: 'var(--color-text-muted)', display: 'flex', justifyContent: 'space-between' }}>
                                        <span>{new Date(project.created_at).toLocaleDateString()}</span>
                                        <span className="mono">{(project.file_size / 1024 / 1024).toFixed(1)} MB</span>
                                    </div>
                                </div>
                            </Link>
                        ))}
                    </div>
                )}
            </div>
        </div>
    )
}
