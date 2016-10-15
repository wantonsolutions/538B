package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"bufio"
	"net"
	t "time"
	gv "github.com/arcaneiceman/GoVector/govec"
)

const (
	SYNCTIME = 1000
	COALESCE = 100
)

const (
	GetTime  = iota
	RespTime = iota
	OffsetTime = iota
	Death = iota
)


var (
	ismaster = flag.Bool("m", false, "summons a master node")
	isslave = flag.Bool("s", false, "summons a slave node")
	masterArgs = "ip:port time d slavesfile logfile"
	slaveArgs = "ip:port time logfile"
	ipPort string
	time int64
	d int64
	slavesfile string
	logfile string

	logger *log.Logger
)

type Msg struct {
	Type int
	Time int64
	Epoch int64
	Offset int64
	Addr *net.UDPAddr
}


func smListen(listen *net.UDPConn, msgChan chan Msg, gvl *gv.GoLog) {
	var message Msg
	buf := make([]byte,1024)
	for true {
		n, addr ,err := listen.ReadFromUDP(buf)
		if err != nil {
			message.Type = Death
			msgChan <- message
			panic(err)
		}
		gvl.UnpackReceive("received time",buf[0:n],&message)
		message.Addr = addr
		msgChan <- message
	}
}

func master(listen *net.UDPConn, time, d int64, slaves map[string]*net.UDPAddr, gvl *gv.GoLog) {
	logger.Printf("starting time %d\n",time)
	var epoch int64

	var message = Msg{Time: time}
	syncTimer := t.After(SYNCTIME *t.Millisecond)
	coalesceTimer := make(<-chan t.Time)
	msgChan := make(chan Msg)
	responses := make(map[string]int64)
	go smListen(listen, msgChan, gvl)

	for true {
		select {
		case m := <-msgChan:
			switch m.Type {
			case RespTime:
				if m.Epoch != epoch {
					logger.Printf("Hey %s Thats an old epoch, catch up you lazy slave!",m.Addr.String())
					break
				}
				_, ok := slaves[m.Addr.String()]
				if !ok {
					logger.Printf("Stop talking %s I don't coordinate slaves not on my roster",m.Addr.String())
					break
				}
				//else it's legitamate
				logger.Printf("Thank You for the response %s\n",m.Addr.String())
				responses[m.Addr.String()] = m.Time
				break

			case GetTime, OffsetTime, Death:
				logger.Printf("Speak when spoken too, %s\n",m.Addr.String())
				break
			}
			break
		case <-syncTimer:
			logger.Printf("Slaves what time is it?\n")
			//increase the epoch and send out get Time requests
			epoch++
			message.Type = GetTime
			message.Time = 0
			message.Epoch = epoch
			broadcast(listen,message,slaves,gvl)
			coalesceTimer = t.After(COALESCE *t.Millisecond)
			break
		case <- coalesceTimer:
			time++
			logger.Printf("Your all wrong the time is %d\n",time)
			message.Type = OffsetTime
			message.Epoch = epoch
			message.Offset = time
			broadcast(listen,message,slaves,gvl)
			syncTimer = t.After(SYNCTIME *t.Millisecond)
			break
		}
	}

	return
}

func broadcast(conn *net.UDPConn, message Msg, slaves map[string]*net.UDPAddr, gvl *gv.GoLog) {
	buf := gvl.PrepareSend("Broadcasting time ",message)
	for _, slave := range slaves {
		conn.WriteToUDP(buf,slave)
	}
}


func slave(conn *net.UDPConn, time int64 , gvl *gv.GoLog) {
	logger.Printf("starting time %d\n",time)
	msgChan := make(chan Msg)

	go smListen(conn, msgChan, gvl)
	for true {
		select {
		case m := <- msgChan:
			switch m.Type {
			case GetTime:
				logger.Printf("Master the time is %d\n",time)
				msg := Msg{Type: RespTime, Time: time, Epoch: m.Epoch, Addr: nil}
				buf := gvl.PrepareSend("Master here is the time",msg)
				_, err := conn.WriteToUDP(buf,m.Addr)
				if err != nil {
					logger.Panic(err)
				}
			case RespTime:
				logger.Printf("Shh %s I'm not allowed to talk to other slaves\n",m.Addr.String())
				break
			case OffsetTime:
				logger.Printf("Sorry Master I'll fix my time by %d\n",m.Offset)
				time = time + m.Offset
				break
			case Death:
				break
			}
		}
	}
		
	return
}



func main () {
	flag.Parse()
	logger = log.New(os.Stdout,"[Launching] ",log.Lshortfile)
	if *ismaster && *isslave {
		logger.Fatal("One can not be both a master and a slave")
	} else if *ismaster {
		startMaster()
	} else if *isslave {
		startSlave()
	} else {
		logger.Fatal("A node with out a lot in life is not a node")
	}
}

func startMaster() {
	if len(os.Args) != 7 {
		logger.Fatal("Masters expect 6 command line arguments: " + masterArgs+ " : passed ",os.Args[1:])
	}
	ipPort = os.Args[2]
	time = stoi(os.Args[3])
	d = stoi(os.Args[4])
	slavesfile = os.Args[5]
	logfile = os.Args[6]
	
	listen := listenConnection(ipPort)
	//setup connections
	ips := readSlaveFile(slavesfile)
	slaves := make(map[string]*net.UDPAddr,len(ips))
	for _, ip := range ips {
		slaves[ip] = getAddr(ip)
	}
	name := "[Master "+ipPort+"] "
	gvl := gv.Initialize(name,logfile)
	//TODO work in govector
	logger.SetPrefix(name)
	master(listen, time, d, slaves, gvl)
	return
}


func startSlave() {
	if len(os.Args) != 5 {
		logger.Fatal("Slaves expect 4 command line arguments: " + slaveArgs + " : passed ",os.Args[1:])
	}
	ipPort = os.Args[2]
	time = stoi(os.Args[3])
	//TODO work in govector
	logfile = os.Args[4]
	listen := listenConnection(ipPort)
	name := "[Slave "+ipPort+"] "
	gvl := gv.Initialize(name,logfile)
	//TODO work in govector
	logger.SetPrefix(name)
	slave(listen, time, gvl)
}

func stoi(time string) int64 {
	t, err := strconv.Atoi(time)
	if err != nil {
		logger.Fatal(err)
	}
	return int64(t)
}

func listenConnection(ip string) *net.UDPConn {
	lAddr, err := net.ResolveUDPAddr("udp", ip)
	if err != nil {
		logger.Fatal(err)
	}
	listen, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		logger.Fatal(err)
	}
	return listen

}

func getAddr(ip string) *net.UDPAddr {
	rAddr, errR := net.ResolveUDPAddr("udp", ip)
	if errR != nil {
		logger.Fatal(errR)
	}
	return rAddr
}

func readSlaveFile(filename string) []string {
	f, err := os.Open(filename)
	if err != nil {
		logger.Fatal(err)
	}
	defer f.Close()
	var ips []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ips = append(ips, scanner.Text())
	}
	if scanner.Err() != nil {
		logger.Fatal(scanner.Err())
	}
	return ips
}

