#!/bin/bash


#go run node.go -b localhost:19000 3000 80 1000 1000 a a 
#args
#node bootstrap rtt FlipProb FlipInvoikeCS CSSleepTime shivizlog dinvlog

#gnome-terminal -x go run node.go -b localhost:19000 200 100 1000 100 a a
#read join
#gnome-terminal -x go run node.go -j localhost:19000 localhost:19002 200 100 1000 100 c c
#gnome-terminal -x go run node.go -b localhost:19001 200 100 1000 100 b b



#go run node.go -b localhost:19000 200 100 1000 100 a a &
#go run node.go -b localhost:19001 200 100 1000 100 b b &
#go run node.go -b localhost:19002 200 100 1000 100 c c &
case "$1" in
    # a single join
    1)
        go run node.go -b localhost:19000 50 100 1000 10 a a &
        sleep 2
        go run node.go -j localhost:19000 localhost:19001 50 100 1000 10 b b &
        ;;
    # double join same head
    2)
        go run node.go -b localhost:19000 50 100 1000 10 a a &
        sleep 2
        go run node.go -j localhost:19000 localhost:19001 50 100 1000 10 b b &
        sleep 5
        go run node.go -j localhost:19000 localhost:19002 50 100 1000 10 c c &
        ;;
    # double join different head
    3)
        go run node.go -b localhost:19000 50 100 1000 10 a a &
        sleep 2
        go run node.go -j localhost:19000 localhost:19001 50 100 1000 10 b b &
        sleep 5
        go run node.go -j localhost:19001 localhost:19002 50 100 1000 10 c c &
        ;;
    # double concurrent join
    4)
        go run node.go -b localhost:19000 50 100 1000 10 a a &
        sleep 2
        go run node.go -j localhost:19000 localhost:19001 50 100 1000 10 b b &
        go run node.go -j localhost:19000 localhost:19002 50 100 1000 10 c c &
        ;;
    #fast double join, should fail
    5)
        go run node.go -b localhost:19000 50 100 1000 10 a a &
        sleep 2
        go run node.go -j localhost:19000 localhost:19001 50 100 1000 10 b b &
        go run node.go -j localhost:19001 localhost:19002 50 100 1000 10 c c &
        ;;
    #triple concurrent join
    6)
        go run node.go -b localhost:19000 50 100 1000 10 a a &
        sleep 2
        go run node.go -j localhost:19000 localhost:19001 50 100 1000 10 b b &
        sleep 5
        go run node.go -j localhost:19001 localhost:19002 50 100 1000 10 c c &
        sleep 5
        go run node.go -j localhost:19000 localhost:19003 50 100 1000 10 d d &
        go run node.go -j localhost:19001 localhost:19004 50 100 1000 10 e e &
        go run node.go -j localhost:19002 localhost:19005 50 100 1000 10 f f &
        ;;
    #triple concurrent join interactive
    7)
        gnome-terminal -x go run node.go -b localhost:19000 50 100 1000 10 a a 
        sleep 2
        gnome-terminal -x go run node.go -j localhost:19000 localhost:19001 50 100 1000 10 b b 
        sleep 5
        gnome-terminal -x go run node.go -j localhost:19001 localhost:19002 50 100 1000 10 c c 
        sleep 5
        gnome-terminal -x go run node.go -j localhost:19000 localhost:19003 50 100 1000 10 d d 
        gnome-terminal -x go run node.go -j localhost:19001 localhost:19004 50 100 1000 10 e e 
        gnome-terminal -x go run node.go -j localhost:19002 localhost:19005 50 100 1000 10 f f 
        ;;
    #single join
    -c)
        rm *.txt
        rm *.dtrace
        rm *.gz
        ;;
    *)
        go run node.go -b localhost:19000 50 100 1000 10 a a &
        sleep 2
        go run node.go -j localhost:19000 localhost:19001 50 100 1000 10 b b &
        ;;
esac
