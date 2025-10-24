**Repository Description**
This software is under heavy development.

🔍 **RTSP/Video Stream AV Sync Analyzer** – A diagnostic tool for detecting and analyzing audio-video synchronization issues in live or recorded video streams (e.g., RTSP, MP4, MKV).

This Go-based utility helps identify:
- **Start time misalignment** between audio and video streams
- **PTS (Presentation Timestamp) drift** over time
- **Average desynchronization** across frames
- **Fixed offsets** vs. progressive drift


- Supports everything that FFmpeg does;
- CSV input for batch processing multiple cameras (with `name`, `uri`, `apartment`)
- Multiple analysis modes:
  - `startdiff`: Compare stream start times from container metadata
  - `firstpackets`: Check alignment of first decoded audio/video frames
  - `trackdiff`: Compute average PTS difference across frames
  - `drift`: Detect progressive desync (clock drift) between streams

###  Use Cases
- Debugging lip-sync issues in surveillance or broadcast systems
- Validating encoder/streaming pipeline integrity
- Monitoring long-term AV sync stability
- QA testing for video ingestion services

###  Dependencies
- `ffmpeg` and `ffprobe` (must be in `$PATH`)
- Go 1.20+
```
Usage: find_desync [-h|--help] -f|--file "<value>" [-p|--packets <integer>]
                   [-t|--time <integer>] -m|--mode "<value>" -s|--subject
                   <integer>

```
Arguments:


*  -h  --help     Print help information
*  -f  --file     File/stream to analyze
*  -p  --packets  Number of packets  to analyze. Default: 10
*  -t  --time     Time of the input to analyze. Default: 10
*  -m  --mode     Mode to analyze: trackdiff, drift, firstpackets,startdiff
*  -s  --subject  Analyze directly source, or save it's slice to the temp file before
