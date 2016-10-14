#!/bin/bash

TIME=0
D=0
LOGNAME=VectorLog.log


go run node.go -m 127.0.0.1:6000 $TIME $D slaves.txt $LOGNAME

while IFS='' read -r line || [[ -n "$line" ]]; do
    go run node.go -s $line $TIME $LOGNAME
done < slaves.txt
