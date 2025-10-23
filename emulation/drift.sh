#!/bin/bash

ffmpeg -stream_loop -1 -i av_sync_test.mp4 \
       -af "atempo=1.05" \
       -c:v libx264 -c:a aac -f mpegts udp://localhost:1234
