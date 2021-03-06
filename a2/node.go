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
	INCTIME = 10
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

type ClockSync struct {
	Node *net.UDPAddr
	Time int64
	Sent t.Time
	Received t.Time
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
	syncTimer := t.After(t.Duration(d * SYNCTIME) *t.Millisecond)
	coalesceTimer := make(<-chan t.Time)
	incTimer := t.After(t.Duration(INCTIME) *t.Millisecond)
	msgChan := make(chan Msg)
	sync := make(map[string]ClockSync)
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
				//logger.Printf("Thank You for the response %s\n",m.Addr.String())
				cs := sync[m.Addr.String()]
				cs.Received = t.Now()
				cs.Time = m.Time
				sync[m.Addr.String()] = cs
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
			sync = make(map[string]ClockSync)
			message.Type = GetTime
			message.Time = 0
			message.Epoch = epoch
			//broadcast to nodes

			buf := gvl.PrepareSend("Broadcasting time ",message)
			for _, slave := range slaves {
				now := t.Now()
				//set send and receive to same value and check later
				//if a message was actually received
				var cs ClockSync
				cs.Node = slave
				cs.Sent = now
				cs.Received = now
				sync[slave.String()] = cs
				listen.WriteToUDP(buf,slave)
			}
			//logger.Println(sync)
			coalesceTimer = t.After(t.Duration(d *COALESCE) *t.Millisecond)
			break
		case <- coalesceTimer:
			
			total := time
			count := 1
			logger.Printf("Your all wrong the time is %d\n",time)
			for _, cs := range sync {
				if cs.Sent.Equal(cs.Received) {
					logger.Printf("Hey %s whats the deal, I never heard back from you",cs.Node.String())
					continue
				} else {
					total += cs.Time
					count ++
				}
			}
			time = total / int64(count)
			for _, cs := range sync {
				if cs.Sent.Equal(cs.Received) {
					continue
				} else {
					message.Type = OffsetTime
					message.Epoch = epoch
					rtt := cs.Received.Sub(cs.Sent).Nanoseconds()/1000000/INCTIME
					//logger.Printf("round trip time: %d\n",rtt)
					message.Offset = time - (int64(cs.Time) + rtt)
					//logger.Printf("%s offset your time by %d",s,message.Offset)
					buf := gvl.PrepareSend("Broadcasting time ",message)
					listen.WriteToUDP(buf,cs.Node)
				}
			}
			syncTimer = t.After(t.Duration(d * SYNCTIME) *t.Millisecond)
			break
		case <-incTimer:
			time++
			incTimer = t.After(t.Duration(INCTIME) *t.Millisecond)
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

	incTimer := t.After(INCTIME *t.Millisecond)
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
				logger.Printf("Sorry Master I'll fix my time by %d now %d\n",m.Offset,time+m.Offset)
				time = time+m.Offset
				break
			case Death:
				break
			}
		case <-incTimer:
			time++
			incTimer = t.After(t.Duration(INCTIME) *t.Millisecond)
			break
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

