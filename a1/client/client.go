
package main

import (
	"net"
	"flag"
	"log"
	"os"
)

const BUFFLEN = 1024

var (
	sPort = flag.String("sport", "11235", "port that the client will send on")
	sip = flag.String("sip", "198.162.52.146", "ip the server will be available at")
	logger *log.Logger
)

func main() {
	flag.Parse()
	conn := setupConnection(*sPort,*sip)
	logger = log.New(os.Stdout,"[Client->"+*sip+":"+*sPort+"] ",log.Lshortfile)
	defer conn.Close()
	buf := make([]byte,1024)
	conn.Write([]byte("fortune"))
	n, err := conn.Read(buf)
	panicError(err)
	logger.Println(string(buf[0:n]))
}

func setupConnection(serverPort, serverIp string) *net.UDPConn {
	rAddr, errR := net.ResolveUDPAddr("udp", serverIp+":"+serverPort)
	panicError(errR)
	conn, errDial := net.DialUDP("udp", nil, rAddr)
 	panicError(errDial)
	if (errR == nil) && (errDial == nil) {
		return conn
	}
	return nil
}

func panicError(err error) {
	if err != nil {
		logger.Panic(err)
	}
}
