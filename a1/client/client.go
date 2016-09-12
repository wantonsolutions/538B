
package main

import (
	"net"
	"flag"
	"log"
	"os"
	"bufio"
)

var (
	sPort = flag.String("sport", "6000", "port that the client will send on")
	lPort = flag.String("lport", "6001", "port that the client will send on")
	sip = flag.String("sip", "127.0.0.1", "ip the server will be available at")
	lip = flag.String("lip", "127.0.0.1", "ip the server will be available at")
	logger *log.Logger
)

func main() {
	flag.Parse()
	logger = log.New(os.Stdout,"[Client "+*lip+":"+*lPort+"] ",log.Lshortfile)
	conn := setupConnection(*sPort,*lPort,*sip,*lip)
	defer conn.Close()

	var (
		buf [1024] byte
	)
	stdin := bufio.NewReader(os.Stdin)
	for true {
		n, _ := stdin.Read(buf[0:])
		if n > 0 {
			conn.Write(buf[0:n])
		}
	}
}

func setupConnection(serverPort, localPort, serverIp, localIp string) *net.UDPConn {
	rAddr, errR := net.ResolveUDPAddr("udp", serverIp+":"+serverPort)
	panicError(errR)
	lAddr, errL := net.ResolveUDPAddr("udp", localIp+":"+localPort)
	panicError(errL)

	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
 	panicError(errDial)
	if (errR == nil) && (errL == nil) && (errDial == nil) {
		return conn
	}
	return nil
}

func panicError(err error) {
	if err != nil {
		logger.Panic(err)
	}
}
