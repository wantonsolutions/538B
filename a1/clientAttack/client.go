
package main

import (
	"net"
	"flag"
	"log"
	"time"
	"fmt"
)

const ( 
	BUFFLEN = 1024
	THREADS = 128
)

var (
	sPort = flag.String("sport", "11235", "port that the client will send on")
	sip = flag.String("sip", "198.162.52.146", "ip the server will be available at")
	logger *log.Logger
)

func main() {
	flag.Parse()
	cpus := THREADS
	bytes := 0
	ping := make([]int64,cpus)
	for i := range ping {
		go func (thread int) {
			conn := setupConnection(*sPort,*sip)
			defer conn.Close()
			buf := make([]byte,1024)
			timer := time.Now()
			for {
				now := time.Now()
				ping[thread] = now.Sub(timer).Nanoseconds()
				timer = now
				conn.Write([]byte("fortune"))
				n, _ := conn.Read(buf)
				bytes += n

			}
		} (i)
	}
	for {
		var total int64
		for i := range ping {
			total += ping[i]
		}
		fmt.Printf("\rthreads %d ping %d bytes %d",cpus ,total/int64(cpus),bytes)
	}

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
