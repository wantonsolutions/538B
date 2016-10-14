#!/bin/bash

TIME=0
D=0


go run node.go -m 127.0.0.1:6000 $TIME $D slaves.txt master &

while IFS='' read -r line || [[ -n "$line" ]]; do
    go run node.go -s $line $TIME $line &
done < slaves.txt
