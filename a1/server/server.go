package main

import (
	"net"
	"flag"
	"log"
	"os"
)

var (
	port = flag.String("port", "6000", "port that the server will listen on")
	ip = flag.String("ip", "127.0.0.1", "ip the server will be available at")
	logger *log.Logger
)

func main() {
	flag.Parse()
	logger = log.New(os.Stdout,"[Server "+*ip+":"+*port+"]",log.Lshortfile)
	saddr, err := net.ResolveUDPAddr("udp", *ip+":"+*port)
	if err != nil {
		logger.Panic(err)
	}
	conn, err := net.ListenUDP("udp",saddr)
	if err != nil {
		logger.Panic(err)
	}
	defer conn.Close()
	var buf  [1024]byte
	for {
		n, addr, err := conn.ReadFrom(buf[0:])
		if err != nil {
			logger.Panic(err)
		}
		logger.Printf("From [%s]: %s",addr.String(),buf[0:n])
	}
}





