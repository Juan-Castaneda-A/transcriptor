import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { supabase } from './services/supabase'
import { wsClient } from './services/websocket'

// Pages
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Upload from './pages/Upload'
import Transcription from './pages/Transcription'

function App() {
    const [session, setSession] = useState(null)
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        // Check initial session
        supabase.auth.getSession().then(({ data: { session } }) => {
            setSession(session)
            if (session?.user) {
                localStorage.setItem('access_token', session.access_token)
                wsClient.connect(session.user.id)
            }
            setLoading(false)
        })

        // Listen for auth changes
        const { data: { subscription } } = supabase.auth.onAuthStateChange((_event, session) => {
            setSession(session)
            if (session?.user) {
                localStorage.setItem('access_token', session.access_token)
                wsClient.connect(session.user.id)
            } else {
                localStorage.removeItem('access_token')
                wsClient.disconnect()
            }
        })

        return () => subscription.unsubscribe()
    }, [])

    if (loading) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
            </div>
        )
    }

    return (
        <BrowserRouter>
            <Routes>
                <Route
                    path="/login"
                    element={!session ? <Login /> : <Navigate to="/" />}
                />
                <Route
                    path="/"
                    element={session ? <Dashboard /> : <Navigate to="/login" />}
                />
                <Route
                    path="/upload"
                    element={session ? <Upload /> : <Navigate to="/login" />}
                />
                <Route
                    path="/transcription/:id"
                    element={session ? <Transcription /> : <Navigate to="/login" />}
                />
                <Route path="*" element={<Navigate to="/" />} />
            </Routes>
        </BrowserRouter>
    )
}

export default App
