#!/bin/bash


#go run node.go -b localhost:19000 3000 80 1000 1000 a a 



gnome-terminal -x go run node.go -b localhost:19000 200 80 1000 1000 a a 
gnome-terminal -x go run node.go -b localhost:19001 200 80 1000 1000 b b
gnome-terminal -x go run node.go -b localhost:19002 200 80 1000 1000 c c
