package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"bufio"
	"net"
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

func master(listen *net.UDPConn, time, d int64, slaves map[string]*net.UDPConn) {
	logger.Println("Starting")
	return
}


func slave(conn *net.UDPConn, time int64) {
	logger.Println("Starting")
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
	
	lAddr, err := net.ResolveUDPAddr("udp", ipPort)
	if err != nil {
		logger.Fatal(err)
	}
	listen, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		logger.Fatal(err)
	}

	//setup connections
	ips := readSlaveFile(slavesfile)
	slaves := make(map[string]*net.UDPConn,len(ips))
	for _, ip := range ips {
		slaves[ip] = setupConnection(ip)
	}
	//TODO work in govector
	logger.SetPrefix("[Master "+ipPort+"] ")
	master(listen, time, d, slaves)
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
	conn := setupConnection(ipPort)
	logger.SetPrefix("[Slave "+ipPort+"] ")
	slave(conn, time)
}

func stoi(time string) int64 {
	t, err := strconv.Atoi(time)
	if err != nil {
		logger.Fatal(err)
	}
	return int64(t)
}

func setupConnection(ip string) *net.UDPConn {
	rAddr, errR := net.ResolveUDPAddr("udp", ip)
	if errR != nil {
		logger.Fatal(errR)
	}
	conn, errDial := net.DialUDP("udp", nil, rAddr)
	if errDial != nil {
		logger.Fatal(errDial)
	}
	return conn
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
