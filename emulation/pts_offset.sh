#!/bin/bash
ffmpeg -re -stream_loop -1 -i av_sync_test.mp4  -c copy -bsf:v setts=pts=PTS+5/TB   -f mpegts udp://localhost:1234
