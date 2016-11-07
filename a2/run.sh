#!/bin/bash

TIME=0
D=1



while IFS='' read -r line || [[ -n "$line" ]]; do
    gnome-terminal -x go run node.go -s $line $TIME $line &
done < slaves.txt
go run node.go -m 127.0.0.1:6000 $TIME $D slaves.txt master &

#go run node.go -s 127.0.0.1:6001 500 A &
#go run node.go -s 127.0.0.1:6002 50000 B &
#go run node.go -s 127.0.0.1:6003 5034530 C &
#go run node.go -s 127.0.0.1:6004 5 D &
#go run node.go -m 127.0.0.1:6005 570 E &
