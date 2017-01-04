package main

import (
	"os"
	"log"
	"strconv"
	"net"
	"math/rand"
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
	SLEEP = iota
	ADDPEER = iota
)

const (
	AYA = iota
	IAA = iota
	JOIN = iota
	JOIN_ACK = iota
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
	csqueue *FuncQueue
	lamport int
	joined bool
	critical = false

	gvl *gv.GoLog

	pl = []string{"localhost:19000","localhost:19001","localhost:19002"}

	debug = true

)

type Peer struct {
	Addr *net.UDPAddr
	Status int
}

type Msg struct {
	Type int
	Lamport int
	Addr *net.UDPAddr
	NewNode string //ip:port
	Cluster []string //used to announce the cluster to a new node
}

func main() {
	Init()
	logger.SetPrefix("["+ipPort+"] ")
	//timers and structures
	//msgChan is a channel for incomming messages to be recived on
	msgChan := make(chan Msg)
	//The heartbeat timer is run periodically to check for the
	//livelyness of peers
	var HEARTBEAT = rtt * 5
	heartbeat := t.After(t.Duration(HEARTBEAT) *t.Millisecond)
	//death tracks if a peer has died, alive[peer] = false until they
	//respond
	var alive map[string]bool
	death := make(<-chan t.Time)
	//flipInvodeCS is the call to try and enter the critical section
	invokeCS := t.After(t.Duration(flipInvokeCS) *t.Millisecond)
	//retry causes a node to send all messages to itself, this is for
	//progression.
	var RETRY = HEARTBEAT * 5
	retry := t.After(t.Duration(RETRY) *t.Millisecond)
	var replied map[string]bool
	var withheld map[string]bool
	var requestLamport int
	//keep track of two phas commit
	var acked_prepare map[string]bool
	var acked_commit map[string]bool

	//keep track of the node which requested to join
	var joiningNode string


	go Listen(conn, msgChan, gvl)

	if ! joined {
		join(conn,gvl,msgChan,heartbeat,death)
	}
	
	for true {
		dinvRT.Dump(ipPort+"117","critical",critical)
		select {
		//Case where a new message has been received
		case m := <-msgChan:
			switch m.Type {
			//Received a heartbeat
			case AYA:
				msg := Msg{IAA,lamport,nil,"",nil}
				send(msg,m.Addr,"I am alive")
				break
			//Received an are you alive
			case IAA:
				//logger.Printf("%s is alive\n",m.Addr.String())
				alive[m.Addr.String()] = true
				break
			//Was queried for the critical section
			case CRITICAL_QUERY:
				//logger.Printf("My Lamport = %d Your lamport = %d\n",requestLamport,m.Lamport)
				if critical {
					logger.Println("in the critical so witholding the critical")
					withheld[m.Addr.String()] = true
				} else if csqueue.Length() == 0 {
					lamportAlg(m.Lamport)
					msg := Msg{CRITICAL_ACK,lamport,nil,"",nil}
					send(msg,m.Addr,"Sure take the critical")
				} else if (m.Lamport < requestLamport) {
					lamportAlg(m.Lamport)
					msg := Msg{CRITICAL_ACK,lamport,nil,"",nil}
					send(msg,m.Addr,"I want to execute the critical but you lamport is lower")
				} else if (m.Lamport == requestLamport && m.Addr.String() < ipPort) {
					//TODO this is incorrect write a function to check
					//this 
					lamportAlg(m.Lamport)
					msg := Msg{CRITICAL_ACK,lamport,nil,"",nil}
					send(msg,m.Addr,"I want to execute the critical, we tied lamports but your ip:port won, take the critical")
				} else {
					//predicate for this section is that you have a
					//critical enqueue, and this peer needs to wait
					//logger.Println("withholding the critical")
					withheld[m.Addr.String()] = true
				}
				break
			case CRITICAL_ACK:
				canexecute := true
				replied[m.Addr.String()] = true
				for peer := range Peers {
					//logger.Printf("checking ack for peer %s\n",peer)
					if !replied[peer] {
						//logger.Printf("Node %s Has not acked\n",peer)
						canexecute = false
						break
					}
				}
				if canexecute {
					//logger.Println("Executing Critical")
					//get into the critical section
					f := csqueue.Peek()
					switch f.Type {
					case SLEEP:
						critical = true
						dinvRT.Dump(ipPort+"171","critical",critical)
						SleepCS()
						critical = false
						//post critical send acks to withheld peers
						msg := Msg{CRITICAL_ACK,lamport,nil,"",nil}
						for name, peer := range Peers {
							if withheld[name] {
								send(msg,peer.Addr,"Sorry for making you wait buddy here is the critical ack")
							}
						}
						csqueue.Remove()
						break
					//we are now in the add peer mode
					case ADDPEER:
						critical = true
						acked_prepare = make(map[string]bool,0)
						msg := Msg{PREPARE,lamport,nil,f.Node,nil}
						broadcast(msg,"Do you ladys mind if "+f.Node+" joins?")
						joiningNode = f.Node
						logger.Println("Broadcasting joins")
						break
					default:
						//logger.Println("DOING NOTHING IN THE CRITICAL")
						break
					}
					invokeCS = t.After(t.Duration(flipInvokeCS) *t.Millisecond)
				}
			//Two Phase Commit for a new node
			case JOIN:
				//requesting the critical section
				replied = make(map[string]bool,len(Peers))
				withheld = make(map[string]bool,len(Peers))
				for name  := range Peers {
					withheld[name] = false
					replied[name] = false
				}
				csqueue.Add(Job{ADDPEER,m.Addr.String()})
				lamport++
				requestLamport = lamport
				msg := Msg{CRITICAL_QUERY,requestLamport,nil,"",nil}
				broadcast(msg,"Requesting the critical section to join")
				break
			case PREPARE:
				addPeer(m.NewNode,Peers)
				p := Peers[m.NewNode]
				p.Status = Proposed
				Peers[m.NewNode] = p
				msg := Msg{PREPARE_ACK,lamport,nil,m.NewNode,nil}
				send(msg,m.Addr,"Sure im ok with "+m.NewNode+" joining")
				break
			case PREPARE_ACK:
				cancommit := true
				acked_prepare[m.Addr.String()] = true
				for peer := range Peers {
					//logger.Printf("checking ack for peer %s\n",peer)
					if !acked_prepare[peer] {
						cancommit = false
						break
					}
				}
				if cancommit {
						acked_commit = make(map[string]bool,0)
						msg := Msg{COMMIT,lamport,nil,m.NewNode,nil}
						broadcast(msg,"Okay ladys "+m.NewNode+" is part of the group now?")
				}
				break
			case COMMIT:
				p := Peers[m.NewNode]
				p.Status = Member
				Peers[m.NewNode] = p
				msg := Msg{COMMIT_ACK,lamport,nil,m.NewNode,nil}
				send(msg,m.Addr,"Ok "+m.NewNode+" you are part of the cluster")
			case COMMIT_ACK:
				canjoin := true
				acked_commit[m.Addr.String()] = true
				for peer := range Peers {
					//logger.Printf("checking ack for peer %s\n",peer)
					if !acked_commit[peer] {
						canjoin = false
						break
					}
				}
				if canjoin {
						peers := peersToString(Peers)
						msg := Msg{JOIN_ACK,lamport,nil,"",peers}
						//TODO FOR THE START RETRY IS NOT WORKING
						send(msg,getAddr(m.NewNode),"Come on peer get in here")
						//release the critical section
						msg = Msg{CRITICAL_ACK,lamport,nil,"",nil}
						for name, peer := range Peers {
							if withheld[name] {
								send(msg,peer.Addr,"Sorry for making you wait buddy here is the critical ack")
							}
						}
						csqueue.Remove()
						addPeer(m.NewNode,Peers)
						joiningNode = ""
						death = nil
						heartbeat = t.After(t.Duration(HEARTBEAT) *t.Millisecond)

						critical = false
				}
				break
			}

			
			break
		


		//Handling a local timeout
		case <-heartbeat:
			alive = make(map[string]bool,len(Peers))
			for name  := range Peers {
				alive[name] = false
			}
			msg := Msg{AYA,lamport,nil,"",nil}
			broadcast(msg,"Are you alive?")
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
			break
		case <-invokeCS:
			if rand.Float32() > flipProb {
				invokeCS = t.After(t.Duration(flipInvokeCS) *t.Millisecond)
				//dont try to get in the critical section
				break
			}
			replied = make(map[string]bool,len(Peers))
			withheld = make(map[string]bool,len(Peers))
			for name  := range Peers {
				withheld[name] = false
				replied[name] = false
			}
			csqueue.Add(Job{SLEEP,""})
			lamport++
			requestLamport = lamport
			msg := Msg{CRITICAL_QUERY,requestLamport,nil,"",nil}
			broadcast(msg,"Requesting the critical section to sleep")
			break
		case <-retry:
			self := getAddr(ipPort)
			//send self a critical ack in case someone died before
			//crit acking
			for peer := range Peers {
				if !debug {
					logger.Printf("peer : %s\n",peer)
				}
			}
			if csqueue.Length() > 0 && !critical {
				msg := Msg{CRITICAL_ACK,lamport,nil,"",nil}
				send(msg,self,"Did anyone die while i was collecting the critical")
				//retry for the critical
				msg = Msg{CRITICAL_QUERY,requestLamport,nil,"",nil}
				broadcast(msg,"Requesting the critical section to sleep")
			}
			if critical && acked_prepare != nil{
				msg := Msg{PREPARE_ACK,lamport,nil,joiningNode,nil}
				send(msg,self,"Did anyone die while joining?")
			}
			if critical && acked_commit != nil{
				msg := Msg{COMMIT_ACK,lamport,nil,joiningNode,nil}
				send(msg,self,"Did anyone die after commiting?")
			}

			retry = t.After(t.Duration(RETRY) *t.Millisecond)
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
	if !debug {
		logger.Println(logMsg)
	}
	_, err := capture.WriteToUDP(conn.WriteToUDP,buf,addr)
	if err != nil {
		panic(err)
	}
}

func lamportAlg(l int){
	if l > lamport {
		lamport = l + 1
	} else {
		lamport++
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
		joined = false
		otheripPort = os.Args[arg]
		arg++
	} else {
		joined = true
	}
	ipPort = os.Args[arg]; arg++
	rtt = stoi(os.Args[arg]); arg++
	flipProb = float32(stoi(os.Args[arg]))/100; arg++ //TODO ask ivan about format
	flipInvokeCS = stoi(os.Args[arg]); arg++
	csSleepTime = stoi(os.Args[arg]); arg++
	gvl = gv.Initialize(ipPort,os.Args[arg]); arg++
	dinvRT.Initalize(ipPort)
	conn = listenConnection(ipPort)
	Peers = make(map[string]Peer)
	csqueue = NewQueue()
	
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

func join(listen *net.UDPConn, gvl *gv.GoLog, msgChan chan Msg, heartbeat, death <-chan t.Time) {
	//make and send a join message to a joiner
	jaddr := getAddr(otheripPort)
	msg := Msg{JOIN,lamport,nil,"",nil}
	send(msg,jaddr,"Please Let Me join")
	incluster := false
	joinerAlive := true
	for !incluster {
		select {
		case m := <-msgChan:
			switch m.Type {
			//Received I am alive from the joiner
			case IAA:
				//logger.Printf("%s is alive\n",m.Addr.String())
				joinerAlive = true
				break
			//Was queried for the critical section
			case JOIN_ACK:
				Peers = getPeers(m.Cluster)
				addPeer(m.Addr.String(),Peers)
				incluster = true
				break
			case JOIN:
				logger.Println("%s tring to join, but I'm not in the cluster, i'll play dead\n",m.Addr.String())
				break
			}
		case <-heartbeat:
			joinerAlive = false
			msg := Msg{AYA,lamport,nil,"",nil}
			broadcast(msg,"Are you alive?")
			send(msg,jaddr,"Joiner are you alive?")
			death = t.After(t.Duration(rtt) *t.Millisecond)
			break
		case <- death:
			if !joinerAlive {
				logger.Fatal("The joiner died before I could join the group. Now I must commit Seppuku\n")
			}
			heartbeat = t.After(t.Duration(rtt*5) *t.Millisecond)
			break
		}
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

func addPeer(peer string, peers map[string]Peer){
	peers[peer] = getPeer(peer)
}

func getPeers(peerList []string) map[string]Peer {
	P := make(map[string]Peer,0)
	for i := range peerList {
		if peerList[i] == ipPort {
			continue
		}
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

func peersToString(peers map[string]Peer) (peerstring []string) {
	for peer := range peers {
		peerstring = append(peerstring,peer)
	}
	return peerstring
}

func stoi(time string) int {
	t, err := strconv.Atoi(time)
	if err != nil {
		logger.Fatal(err)
	}
	return t
}

//critical sections
func SleepCS() {
		logger.Println("sleeping")
		t.Sleep(t.Duration(csSleepTime)* t.Millisecond)
		logger.Println("awake")
}

type Job struct {
	Type int
	Node string
}

type FuncQueue struct {
	fqueue []Job
}

func NewQueue() *FuncQueue {
	fq := new(FuncQueue)
	fq.fqueue = make([]Job,0)
	return fq
}

func (q *FuncQueue) Length() int {
	return len(q.fqueue)
}

func (q *FuncQueue) Peek() Job {
	if q.Length() > 0 {
		return q.fqueue[0]
	}
	return Job{-1,""}
}

func (q *FuncQueue) Remove() {
	q.fqueue = q.fqueue[1:]
}

func (q *FuncQueue) Add(f Job) {
	q.fqueue = append(q.fqueue,f)
}
