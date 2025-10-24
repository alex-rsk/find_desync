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

```
*  -h  --help         Print help information
*  -f  --file         File/stream to analyze. No default value.
*  -с  --csv          Annotated CSV file which enumerates sources in format, example :
                      name,uri,apart
                      "Cam1", "rtsp://...", "Apart 1"
*  -p  --packets      Number of packets  to analyze. Mutually exclusive with -t. No default value.
*  -t  --time         Time of the input to analyze. Mutually exclusive with -p. No default value.
*  -m  --method       Method to analyze: trackdiff, drift, firstpackets,startdiff. Default is "startdiff".
*  -d  --direct       Analyze directly source (1), or analyze saved slice of the source (0). Default is 0.
```
