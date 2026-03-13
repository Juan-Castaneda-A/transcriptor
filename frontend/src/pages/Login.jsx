import { useState } from 'react'
import { supabase } from '../services/supabase'

export default function Login() {
    const [loading, setLoading] = useState(false)
    const [email, setEmail] = useState('')
    const [message, setMessage] = useState(null)
    const [error, setError] = useState(null)

    const handleGoogleLogin = async () => {
        setLoading(true)
        setError(null)
        try {
            const { error } = await supabase.auth.signInWithOAuth({
                provider: 'google',
                options: {
                    redirectTo: window.location.origin,
                },
            })
            if (error) throw error
        } catch (err) {
            setError(err.message)
        } finally {
            setLoading(false)
        }
    }

    const handleMagicLink = async (e) => {
        e.preventDefault()
        if (!email) return
        setLoading(true)
        setError(null)
        setMessage(null)
        try {
            const { error } = await supabase.auth.signInWithOtp({
                email,
                options: {
                    emailRedirectTo: window.location.origin,
                },
            })
            if (error) throw error
            setMessage("¡Correo enviado! Revisa tu bandeja de entrada.")
        } catch (err) {
            setError(err.message)
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="page flex items-center justify-center">
            <div className="container" style={{ maxWidth: '440px' }}>
                <div className="card animate-fade-in">
                    <div style={{ textAlign: 'center', marginBottom: '32px' }}>
                        <h1 style={{ fontSize: '2.5rem', marginBottom: '8px' }}>Transcriptor</h1>
                        <p style={{ color: 'var(--color-text-secondary)' }}>
                            Transforma audio en texto con IA.
                        </p>
                    </div>

                    {error && (
                        <div style={{ padding: '12px', background: 'rgba(255, 61, 113, 0.1)', color: 'var(--color-error)', borderRadius: 'var(--radius-md)', marginBottom: '24px', fontSize: '0.875rem' }}>
                            {error}
                        </div>
                    )}

                    {message && (
                        <div style={{ padding: '12px', background: 'rgba(0, 210, 255, 0.1)', color: 'var(--color-primary)', borderRadius: 'var(--radius-md)', marginBottom: '24px', fontSize: '0.875rem' }}>
                            {message}
                        </div>
                    )}

                    <form onSubmit={handleMagicLink} style={{ marginBottom: '24px' }}>
                        <div style={{ marginBottom: '16px' }}>
                            <label style={{ display: 'block', fontSize: '0.75rem', color: 'var(--color-text-muted)', marginBottom: '8px', fontWeight: 'bold' }}>CORREO ELECTRÓNICO</label>
                            <input
                                type="email"
                                className="btn btn-secondary w-full"
                                style={{ textAlign: 'left', cursor: 'text', border: '1px solid var(--color-border)', color: 'white' }}
                                placeholder="tu@email.com"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                required
                            />
                        </div>
                        <button className="btn btn-primary w-full" disabled={loading} type="submit">
                            {loading ? 'Enviando...' : 'Iniciar con Magic Link'}
                        </button>
                    </form>

                    <div style={{ display: 'flex', alignItems: 'center', margin: '24px 0', color: 'var(--color-text-muted)' }}>
                        <div style={{ flex: 1, height: '1px', background: 'var(--color-border)' }}></div>
                        <span style={{ padding: '0 12px', fontSize: '0.75rem' }}>O TAMBIÉN</span>
                        <div style={{ flex: 1, height: '1px', background: 'var(--color-border)' }}></div>
                    </div>

                    <button
                        className="btn btn-secondary w-full"
                        onClick={handleGoogleLogin}
                        disabled={loading}
                    >
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" style={{ marginRight: '10px' }}>
                            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
                            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
                            <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z" />
                            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.66l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
                        </svg>
                        Google (Requiere Configuración)
                    </button>
                </div>
            </div>
        </div>
    )
}
