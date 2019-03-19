package main

import (
	"epaxos/common"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var VERSION string

type InstState int32
type LeaderState int32
type ChannelID int32

//some global variables
const (
	CHAN_MAX = 100 // the maximum number of channels is 100

)

const (
	PreAccepted   InstState = 0
	PreAcceptedOK InstState = 4
	Accepted      InstState = 1
	Committed     InstState = 2
	Prepare       InstState = 3
	Idle          InstState = 5
	Start         InstState = 6
)

type ChangeStateMsg struct {
	success bool
}

type StatefulInst struct {
	inst  common.Instance
	state InstState
}

type chanPointer struct {
	pointer ChannelID
	mu      sync.Mutex
}

type InstList struct {
	Mu      sync.Mutex
	LogFile *os.File
	Offset  common.InstanceID
	Pending []*StatefulInst // one per InstanceID
}

// the state machine for each instance
type InstanceState struct {
	self int
	// channels for state transitions
	getReq         chan bool
	getPreAcceptOK chan bool
	selectFastPath chan bool
	getAcceptOK    chan bool

	state InstState
}

type EPaxos struct {
	verbose  bool
	self     common.ReplicaID
	lastInst common.InstanceID
	array    []*InstList // one InstList per replica
	data     map[common.Key]common.Value
	probesL  sync.Mutex
	probes   map[int64]chan bool
	udp      *net.UDPConn
	rpc      []chan interface{}
	peers    int64 // number of peers, including itself
	mu       sync.Mutex

	// records which channel is allocated for each instance
	inst2Chan map[common.InstanceID]ChannelID
	//chanHead  chanPointer
	//chanTail  chanPointer

	// bitmap for channels
	freeChan map[ChannelID]bool // true means not available

	// channels for Instance state machines
	innerChan []chan interface{}

	// channels to other servers/replicas
	inbound *chan interface{}
}

func NewEPaxos(nrep int64, rep common.ReplicaID) *EPaxos {
	dir := common.GetEnv("EPAXOS_DATA_PREFIX", "./data/data-")
	buff, err := strconv.ParseInt(common.GetEnv("EPAXOS_BUFFER", "1024"), 10, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	ep := new(EPaxos)
	ep.verbose = common.GetEnv("EPAXOS_DEBUG", "TRUE") == "TRUE"
	ep.self = rep
	ep.lastInst = common.InstanceID(0)
	ep.array = make([]*InstList, nrep)
	ep.rpc = make([]chan interface{}, nrep)
	for i := int64(0); i < nrep; i++ {
		fileName := dir + strconv.FormatInt(i, 10) + ".dat"
		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Print(err)
			return nil
		}
		lst := &InstList{
			Mu:      sync.Mutex{},
			LogFile: file,
			Offset:  0,
			Pending: make([]*StatefulInst, 0),
		}
		ep.array[i] = lst
		ep.rpc[i] = make(chan interface{}, buff)
	}
	ep.inbound = &ep.rpc[ep.self]
	endpoint := common.GetEnv("EPAXOS_LISTEN", "0.0.0.0:23333")
	addr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		log.Println(err)
		return nil
	}
	log.Printf("ListenUDP on %s\n", endpoint)
	ep.udp, err = net.ListenUDP("udp", addr)
	ep.udp.SetWriteBuffer(0)
	if err != nil {
		log.Println(err)
		return nil
	}
	ep.probes = make(map[int64]chan bool)
	err = ep.recoverFromLog()
	if err != nil {
		log.Println(err)
		return nil
	}

	ep.data = make(map[common.Key]common.Value)
	ep.peers = nrep
	ep.inst2Chan = make(map[common.InstanceID]ChannelID)
	ep.freeChan = make(map[ChannelID]bool)

	ep.innerChan = make([]chan interface{}, CHAN_MAX)
	return ep
}

type logWriter struct {
	Id common.ReplicaID
}

func (writer *logWriter) Write(bytes []byte) (int, error) {
	return fmt.Printf(
		"%s #%d %s",
		time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		writer.Id,
		string(bytes),
	)
}

func main() {
	logW := new(logWriter)
	logW.Id = -1
	log.SetFlags(log.Lshortfile)
	log.SetOutput(logW)
	rep, err := strconv.ParseInt(common.GetEnv("EPAXOS_REPLICA_ID", "0"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	logW.Id = common.ReplicaID(rep)

	log.Printf("This is epaxos-server, version %s", VERSION)
	common.InitializeRand()
	nrep, err := strconv.ParseInt(common.GetEnv("EPAXOS_NREPLICAS", "1"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	ep := NewEPaxos(nrep, common.ReplicaID(rep))
	if ep == nil {
		log.Fatal("EPaxos creation failed")
	}

	err = ep.forkUdp()
	if err != nil {
		log.Fatal(err)
	}
}
