**Repository Description**
This software is under heavy development.

🔍 **RTSP/Video Stream AV Sync Analyzer** – A diagnostic tool for detecting and analyzing audio-video synchronization issues in live or recorded video streams (e.g., RTSP, MP4, MKV).

This Go-based utility helps identify:
- **Start time misalignment** between audio and video streams
- **PTS (Presentation Timestamp) drift** over time
- **Average desynchronization** across frames
- **Fixed offsets** vs. progressive drift

### ✨ Features
- Supports **RTSP, local files, and network streams**
- Records short clips automatically for analysis (using `ffmpeg`)
- Uses `ffprobe` to extract precise frame-level PTS data
- Color-coded terminal output for quick visual diagnosis
- CSV input for batch processing multiple cameras (with `name`, `uri`, `apartment`)
- Multiple analysis modes:
  - `startdiff`: Compare stream start times from container metadata
  - `firstpackets`: Check alignment of first decoded audio/video frames
  - `trackdiff`: Compute average PTS difference across frames
  - `drift`: Detect progressive desync (clock drift) between streams

### 🛠️ Use Cases
- Debugging lip-sync issues in surveillance or broadcast systems
- Validating encoder/streaming pipeline integrity
- Monitoring long-term AV sync stability
- QA testing for video ingestion services

### 📦 Dependencies
- `ffmpeg` and `ffprobe` (must be in `$PATH`)
- Go 1.20+

> 💡 Ideal for DevOps, video engineers, and QA teams managing large-scale camera or streaming infrastructures.

---

**Example**:  
```bash
go run main.go cameras.csv 10 drift
```
Analyzes 10 seconds of each camera stream for PTS drift and reports desync severity.

Keep your streams in perfect sync! 🎥🔊
