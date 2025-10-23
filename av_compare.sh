#!/bin/bash

# Compare audio and video PTS
#
#
#  Usage:
#   chmod +x av_compare.sh   
#   ./av_compare rtsp://....
#

NPACKETS=$2

if [ -z "$2" ]; then
NPACKETS=10
fi


if [ -z "$1" ]; then
    echo "Usage: $0 <rtsp_url>"
    exit 1
fi

URL="$1"


AUDIO_FILE=$(mktemp)
VIDEO_FILE=$(mktemp)


cleanup() {
    rm -f "$AUDIO_FILE" "$VIDEO_FILE"
}
trap cleanup EXIT


ffprobe -analyzeduration 1M -probesize 1M -rtsp_transport tcp -select_streams a \
    -show_entries "packet=pts_time,duration_time:stream=index,codec_type" \
    -read_intervals "%+${NPACKETS}" -of csv=p=0 "$URL" > "$AUDIO_FILE" 2>/dev/null &
AUDIO_PID=$!

ffprobe -analyzeduration 1M -probesize 1M -rtsp_transport tcp  -select_streams v \
    -show_entries "packet=pts_time,duration_time:stream=index,codec_type" \
    -read_intervals "%+${NPACKETS}" -of csv=p=0 "$URL" > "$VIDEO_FILE" 2>/dev/null &
VIDEO_PID=$!


wait $AUDIO_PID
AUDIO_EXIT=$?
wait $VIDEO_PID
VIDEO_EXIT=$?


if [ $AUDIO_EXIT -ne 0 ] && [ $VIDEO_EXIT -ne 0 ]; then
    echo "Error: Both ffprobe commands failed"
    exit 1
fi


AUDIO_PTS=()
while IFS= read -r line; do
    AUDIO_PTS+=("$line")
done < <(awk -F',' '{print $1}' "$AUDIO_FILE")

VIDEO_PTS=()
while IFS= read -r line; do
    VIDEO_PTS+=("$line")
done < <(awk -F',' '{print $1}' "$VIDEO_FILE")


MAX_ROWS=${#AUDIO_PTS[@]}
if [ ${#VIDEO_PTS[@]} -gt $MAX_ROWS ]; then
    MAX_ROWS=${#VIDEO_PTS[@]}
fi


printf "%-20s | %-20s\n" "Audio PTS Time" "Video PTS Time"
printf "%-20s-+-%-20s\n" "--------------------" "--------------------"


for ((i=0; i<MAX_ROWS; i++)); do
    AUDIO_VAL="${AUDIO_PTS[$i]:-N/A}"
    VIDEO_VAL="${VIDEO_PTS[$i]:-N/A}"
    printf "%-20s | %-20s\n" "$AUDIO_VAL" "$VIDEO_VAL"
done


echo ""
echo "Summary:"
echo "Audio packets: ${#AUDIO_PTS[@]}"
echo "Video packets: ${#VIDEO_PTS[@]}"

