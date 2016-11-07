package main

import (
	"os"
	"log"
	"strconv"
	"net"
	t "time"
	gv "github.com/arcaneiceman/GoVector/govec"
	"github.com/arcaneiceman/GoVector/capture"
	"bitbucket.org/bestchai/dinv/dinvRT"
)


const (
	Member = iota
	Proposed = iota
)

const (
	AYA = iota
	IAA = iota
	CRITICAL_QUERY = iota
	CRITICAL_ACK = iota
	PREPARE = iota
	PREPARE_ACK = iota
	COMMIT = iota
	COMMIT_ACK = iota
)



var (
	otheripPort string
	ipPort string
	rtt int
	flipProb float32
	flipInvokeCS int
	csSleepTime int
	shivizFilename string
	dinvFilename string
	logger *log.Logger
	conn *net.UDPConn
	Peers map[string]Peer

	gvl *gv.GoLog

	peerList = []string{"localhost:19000","localhost:19001","localhost:19002"}

)

type Peer struct {
	Addr *net.UDPAddr
	Status int
}

type Msg struct {
	Type int
	Addr *net.UDPAddr
	NewNode string //ip:port
}


func main() {
	Init()
	logger.SetPrefix("["+ipPort+"] ")
	msgChan := make(chan Msg)
	var HEARTBEAT = rtt * 5
	heartbeat := t.After(t.Duration(HEARTBEAT) *t.Millisecond)
	var alive map[string]bool
	death := make(<-chan t.Time)

	go Listen(conn, msgChan, gvl)
	
	for true {
		select {
		case m := <-msgChan:
			switch m.Type {
			case AYA:
				msg := Msg{IAA,nil,""}
				send(msg,m.Addr,"I am alive")
				break
			case IAA:
				logger.Printf("%s is alive\n",m.Addr.String())
				alive[m.Addr.String()] = true
			}
		case <- heartbeat:
			msg := Msg{AYA,nil,""}
			broadcast(msg,"Are you alive?")
			alive = make(map[string]bool,len(Peers))
			for name  := range Peers {
				alive[name] = false
			}
			death = t.After(t.Duration(rtt) *t.Millisecond)
			break
		case <- death:
			for peer := range Peers {
				if !alive[peer] {
					logger.Printf("peer %s is dead!!!",peer)
					delete(Peers,peer)
				}
			}
			heartbeat = t.After(t.Duration(HEARTBEAT) *t.Millisecond)
		}
	}

}

func broadcast(msg Msg, logMsg string) {
	for _, peer := range Peers {
		send(msg, peer.Addr, logMsg)
	}
}

func send(msg Msg, addr *net.UDPAddr, logMsg string) {
	buf := gvl.PrepareSend(logMsg,msg)
	logger.Println(logMsg)
	_, err := capture.WriteToUDP(conn.WriteToUDP,buf,addr)
	if err != nil {
		panic(err)
	}
}


func Init() {
	logger = log.New(os.Stdout, "[Initializing] ", log.Lshortfile)
	if len(os.Args) < 7 {
		logger.Fatalf("Not enough command line arguments\n")
	}
	if (os.Args[1] != "-j" && os.Args[1] != "-b") {
		logger.Fatalf("Specify if the node is joining -j or bootstrapping -b")
	}
	arg := 2
	if (os.Args[1] == "-j") {
		otheripPort = os.Args[arg]
		arg++
	}
	ipPort = os.Args[arg]; arg++
	rtt = stoi(os.Args[arg]); arg++
	flipProb = float32(stoi(os.Args[arg]))/100; arg++ //TODO ask ivan about format
	flipInvokeCS = stoi(os.Args[arg]); arg++
	csSleepTime = stoi(os.Args[arg]); arg++
	gvl = gv.Initialize(ipPort,os.Args[arg]); arg++
	dinvRT.Initalize(ipPort)
	conn = listenConnection(ipPort)
	Peers = getPeers()
	
}

func Listen(listen *net.UDPConn, msgChan chan Msg, gvl *gv.GoLog) {
	var message Msg
	buf := make([]byte,1024)
	for true {
		n, addr ,err := capture.ReadFromUDP(listen.ReadFromUDP,buf)
		if err != nil {
			msgChan <- message
			panic(err)
		}
		gvl.UnpackReceive("received time",buf[0:n],&message)
		message.Addr = addr
		msgChan <- message
	}
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

//basic
func getPeers() map[string]Peer {
	P := make(map[string]Peer,0)
	for i:= range peerList {
		peer := getPeer(peerList[i])
		P[peer.Addr.String()] = peer
	}
	return P
}

func getPeer(ip string) Peer {
	addr := getAddr(ip)
	return Peer{addr, Member}
}
func getAddr(ip string) *net.UDPAddr {
	rAddr, errR := net.ResolveUDPAddr("udp", ip)
	if errR != nil {
		logger.Fatal(errR)
	}
	return rAddr
}

	



func stoi(time string) int {
	t, err := strconv.Atoi(time)
	if err != nil {
		logger.Fatal(err)
	}
	return t
}
