"""
Transcriptor Worker - AI Transcription Engine (Faster-Whisper Edition)
Consumes jobs from Redis queue, transcribes audio using Faster-Whisper,
and stores results in Supabase.
"""

import os
import sys
import json
import time
import tempfile
import logging
import redis
import requests
from faster_whisper import WhisperModel
from supabase import create_client, Client
from dotenv import load_dotenv

load_dotenv()

# --- Configuration ---
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
SUPABASE_URL = os.getenv("SUPABASE_URL")
SUPABASE_SERVICE_KEY = os.getenv("SUPABASE_SERVICE_KEY")
WHISPER_MODEL_SIZE = os.getenv("WHISPER_MODEL", "base")
# Computing device for faster-whisper (auto / cpu / cuda)
DEVICE = os.getenv("WHISPER_DEVICE", "auto")
COMPUTE_TYPE = os.getenv("WHISPER_COMPUTE_TYPE", "float32") # auto, float16, int8_float16, int8

QUEUE_NAME = "transcription:jobs"
STATUS_CHANNEL = "transcription:status"

# --- Logging ---
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger("worker")

# --- Clients ---
def get_redis_client():
    return redis.from_url(REDIS_URL, decode_responses=True)

def get_supabase_client() -> Client:
    if not SUPABASE_URL or not SUPABASE_SERVICE_KEY:
        logger.error("❌ SUPABASE_URL and SUPABASE_SERVICE_KEY are missing!")
        sys.exit(1)
    return create_client(SUPABASE_URL, SUPABASE_SERVICE_KEY)

def download_audio(storage_url: str, temp_dir: str) -> str:
    """Download audio from Supabase Storage using the public URL."""
    # Note: We use the public URL if the bucket is public, or a signed URL passed in the job.
    # For MVP, we assume public or the backend provides a valid URL.
    full_url = f"{SUPABASE_URL}/storage/v1/object/public/media/{storage_url}"
    logger.info(f"📥 Downloading audio from: {full_url}")

    # We use requests here because it's simple and efficient for binary streams
    response = requests.get(full_url, stream=True, timeout=120)
    response.raise_for_status()

    ext = os.path.splitext(storage_url)[1] or ".mp3"
    local_path = os.path.join(temp_dir, f"audio{ext}")

    with open(local_path, "wb") as f:
        for chunk in response.iter_content(chunk_size=8192):
            f.write(chunk)

    logger.info(f"✅ Downloaded: {os.path.getsize(local_path) / 1024 / 1024:.1f} MB")
    return local_path


def transcribe_audio(audio_path: str, model: WhisperModel) -> dict:
    """
    Transcribe audio using Faster-Whisper.
    """
    logger.info("🧠 Starting transcription with Faster-Whisper...")
    start_time = time.time()

    # beam_size=5 is standard for good accuracy
    segments, info = model.transcribe(audio_path, beam_size=5)

    language = info.language
    logger.info(f"🌐 Detected language: {language} (probability: {info.language_probability:.2f})")

    results = []
    full_text_parts = []
    
    for segment in segments:
        results.append({
            "start_time": round(segment.start, 2),
            "end_time": round(segment.end, 2),
            "content": segment.text.strip(),
            "speaker_label": "Hablante 1", # Diarization placeholder
        })
        full_text_parts.append(segment.text.strip())

    full_text = " ".join(full_text_parts)
    elapsed = time.time() - start_time
    
    logger.info(
        f"✅ Transcription complete: {len(results)} segments, "
        f"time={elapsed:.1f}s"
    )

    return {
        "language": language,
        "full_text": full_text,
        "word_count": len(full_text.split()),
        "segments": results,
    }


def save_results(supabase: Client, file_id: str, result: dict):
    """Save results using the official Supabase library."""
    transcription_id = f"tr_{int(time.time() * 1000)}"

    # 1. Insert Transcription
    res = supabase.table("transcriptions").insert({
        "id": transcription_id,
        "media_file_id": file_id,
        "full_text": result["full_text"],
        "language": result["language"],
        "word_count": result["word_count"],
    }).execute()
    
    logger.info(f"💾 Transcription saved: {transcription_id}")

    # 2. Insert Segments in batch
    if result["segments"]:
        segments_data = []
        for i, seg in enumerate(result["segments"]):
            segments_data.append({
                "id": f"seg_{transcription_id}_{i}",
                "transcription_id": transcription_id,
                "start_time": seg["start_time"],
                "end_time": seg["end_time"],
                "content": seg["content"],
                "speaker_label": seg["speaker_label"],
            })
        
        # Supabase Python handles batching
        supabase.table("segments").insert(segments_data).execute()
        logger.info(f"💾 {len(segments_data)} segments saved")


def update_status(supabase: Client, rdb, file_id: str, user_id: str, status: str, message: str = "", error: str = ""):
    """Update DB status and notify via Redis Pub/Sub."""
    # Update DB
    data = {"status": status, "updated_at": "now()"}
    if error:
        data["error_msg"] = error
    
    supabase.table("media_files").update(data).eq("id", file_id).execute()

    # Notify Frontend via WebSocket (re-broadcasted by Go server)
    update = json.dumps({
        "file_id": file_id,
        "user_id": user_id,
        "status": status,
        "message": message or status
    })
    rdb.publish(STATUS_CHANNEL, update)


def process_job(supabase: Client, rdb, job: dict, model: WhisperModel):
    file_id = job["file_id"]
    storage_url = job["storage_url"]
    user_id = job["user_id"]

    logger.info(f"📋 Processing job for file: {file_id}")
    update_status(supabase, rdb, file_id, user_id, "processing", "Iniciando descarga...")

    with tempfile.TemporaryDirectory() as temp_dir:
        try:
            # 1. Download
            audio_path = download_audio(storage_url, temp_dir)
            
            # 2. Transcribe
            update_status(supabase, rdb, file_id, user_id, "processing", "Transcribiendo con IA...")
            result = transcribe_audio(audio_path, model)

            # 3. Save
            update_status(supabase, rdb, file_id, user_id, "processing", "Guardando resultados...")
            save_results(supabase, file_id, result)

            # 4. Finish
            update_status(supabase, rdb, file_id, user_id, "completed", "¡Transcripción terminada!")
            logger.info(f"🎉 Job completed: {file_id}")

        except Exception as e:
            logger.error(f"❌ Job failed: {str(e)}")
            update_status(supabase, rdb, file_id, user_id, "error", f"Error: {str(e)}", error=str(e))


def main():
    logger.info("🚀 Transcriptor Worker (Faster-Whisper) starting...")
    
    supabase = get_supabase_client()
    rdb = get_redis_client()
    
    # Pre-load the model
    logger.info(f"📦 Loading model: {WHISPER_MODEL_SIZE} on {DEVICE}...")
    model = WhisperModel(WHISPER_MODEL_SIZE, device=DEVICE, compute_type=COMPUTE_TYPE)
    logger.info("✅ Model loaded")

    logger.info(f"👂 Listening on {QUEUE_NAME}...")
    while True:
        try:
            result = rdb.brpop(QUEUE_NAME, timeout=30)
            if result:
                _, raw_msg = result
                job = json.loads(raw_msg)
                process_job(supabase, rdb, job, model)
        except redis.ConnectionError:
            logger.warning("Redis connection lost, retrying...")
            time.sleep(5)
        except Exception as e:
            logger.error(f"Main loop error: {e}")
            time.sleep(2)

if __name__ == "__main__":
    main()
