# Transcriptor 🎧

Sistema robusto de transcripción de audio basado en IA (Whisper) con arquitectura escalable en Go, Python y React.

## 🏗️ Arquitectura

- **Backend (API Gateway):** Go (Golang) - Maneja auth, subidas, gestión de proyectos y WebSockets.
- **Worker (IA):** Python + OpenAI Whisper - Procesa archivos de audio de forma asíncrona.
- **Frontend:** React + Vite - Editor e interfaz moderna con Look & Feel premium.
- **Base de Datos & Storage:** Supabase (PostgreSQL + S3).
- **Cola de Mensajes:** Redis.

## 🚀 Guía de Inicio Rápido

### 1. Requisitos Previos

- Go 1.22+
- Node.js 20+
- Python 3.10+
- Docker (opcional, para Redis)

### 2. Configuración de Supabase

Crea un proyecto en [Supabase](https://supabase.com/) y ejecuta los siguientes pasos:

1. **Base de Datos:** Crea las tablas ejecutando este SQL:
   ```sql
   -- Media Files
   CREATE TABLE media_files (
     id TEXT PRIMARY KEY,
     user_id UUID REFERENCES auth.users(id),
     storage_url TEXT NOT NULL,
     file_name TEXT NOT NULL,
     file_size BIGINT,
     mime_type TEXT,
     status TEXT DEFAULT 'pending',
     error_msg TEXT,
     created_at TIMESTAMPTZ DEFAULT now(),
     updated_at TIMESTAMPTZ DEFAULT now()
   );

   -- Transcriptions
   CREATE TABLE transcriptions (
     id TEXT PRIMARY KEY,
     media_file_id TEXT REFERENCES media_files(id) ON DELETE CASCADE,
     full_text TEXT,
     language TEXT,
     word_count INT,
     created_at TIMESTAMPTZ DEFAULT now(),
     updated_at TIMESTAMPTZ DEFAULT now()
   );

   -- Segments
   CREATE TABLE segments (
     id TEXT PRIMARY KEY,
     transcription_id TEXT REFERENCES transcriptions(id) ON DELETE CASCADE,
     start_time FLOAT,
     end_time FLOAT,
     speaker_label TEXT,
     content TEXT
   );
   ```

2. **Storage:** Crea un bucket llamado **`media`** y hazlo **público** (o configura políticas RLS).

### 3. Variables de Entorno

Copia el archivo `.env.example` a `.env` en la raíz y llena los datos de Supabase y Redis.

### 4. Ejecución

#### Backend (Go)
```bash
cd backend
go run cmd/main.go
```

#### Worker (Python)
```bash
cd worker
python -m venv venv
# Windows: venv\\Scripts\\activate
pip install -r requirements.txt
python main.py
```

#### Frontend (React)
```bash
cd frontend
npm install
npm run dev
```

## 🛠️ Stack Tecnológico

- **Go:** Gofiber (o std native routes), Gorilla WebSocket.
- **Python:** Faster-Whisper, PyTorch.
- **Frontend:** React, React Router, CSS Variables (Pure CSS).
- **Cola:** Redis (LPUSH/BRPOP).

## 📄 Licencia
Este proyecto está bajo la Licencia MIT.
