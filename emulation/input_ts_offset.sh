#!/bin/bash
ffmpeg -re -stream_loop -1 -i av_sync_test.mp4 -itsoffset 1.5 -i av_sync_test.mp4 -map 0:v -map 1:a -c copy -f mpegts udp://localhost:1234
