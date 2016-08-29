#!/bin/bash

# KGO in SF
#FREQUENCY=177000000
#PROGRAM=3

# KQED in SF
FREQUENCY=569000000
PROGRAM=1

FILENAME=myvid.mp4

rm -f ${FILENAME} encode.log playback.log

cvlc -vvv \
	atsc://frequency=$FREQUENCY \
	:program=$PROGRAM \
	--sout="#transcode{vcodec=mp4v,width=1280,height=720,deinterlace}:std{access=file,mux=ps,dst=${FILENAME}}" \
	--no-sout-all

# log somewhere
#  	--file-logging \
#  	--logfile encode.log

# stop automatically
#  	--start-time 0 \
#  	--run-time 15 \
#  	vlc://quit

# playback
cvlc -vvv \
	${FILENAME} \
	--file-logging \
	--logfile playback.log \
	vlc://quit

## Resources

# See http://www.videolan.org/streaming-features.html

# --no-sout-all: https://forum.videolan.org/viewtopic.php?t=128143#p440105
# --sout-ts-shaping=4000: https://forum.videolan.org/viewtopic.php?t=128143#p440943
#  -not needed
