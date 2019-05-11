package main

import (
	"context"
	"epaxos/common"
	"fmt"
	"golang.org/x/sync/semaphore"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var VERSION string

const (
	Invalid      common.InstState = 0
	PreAccepting common.InstState = 1
	Accepting    common.InstState = 2
	Committing   common.InstState = 3
	Finished     common.InstState = 4
	Prepare      common.InstState = 5
)

type StatefulInst struct {
	inst  common.Instance
	state InstState
}

type InstList struct {
	Mu      sync.RWMutex // one lock for all fields
	LogFile *os.File
	Offset  common.InstanceID
	Pending []*StatefulInst // one per InstanceID
}

type EPaxos struct {
	verbose bool
	self    common.ReplicaID
	data    map[common.Key]common.Value
	probesL sync.Mutex
	probes  map[int64]chan bool
	udp     *net.UDPConn
	rpc     []chan interface{}
	peers   int64 // number of peers, including itself
	timeout time.Duration

	array []*InstList // one InstList per replica

	ctx      context.Context     // Limiting concurrent isms
	sem      *semaphore.Weighted // Limiting concurrent isms
	ismsL    sync.RWMutex        // Restrict access to ep.isms and ep.nextInst
	isms     map[common.InstanceID]*InstStateMachine

    psmBook  map[common.InstRef]*PrepareStateMachine
	nextInst common.InstanceID
}

func NewEPaxos(nrep int64, rep common.ReplicaID) *EPaxos {
	dir := common.GetEnv("EPAXOS_DATA_PREFIX", "./data/data-")
	to, err := strconv.ParseInt(common.GetEnv("EPAXOS_TIMEOUT", "5000"), 10, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	buff, err := strconv.ParseInt(common.GetEnv("EPAXOS_BUFFER", "1024"), 10, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	rbuff, err := strconv.ParseInt(common.GetEnv("EPAXOS_UDP_BUFFER_READ", "26214400"), 10, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	wbuff, err := strconv.ParseInt(common.GetEnv("EPAXOS_UDP_BUFFER_WRITE", "1048576"), 10, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	pipe, err := strconv.ParseInt(common.GetEnv("EPAXOS_PIPE", "512"), 10, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	ep := new(EPaxos)
	ep.verbose = common.GetEnv("EPAXOS_DEBUG", "TRUE") == "TRUE"
	log.Printf("I'm #%d, total %d replicas", rep, nrep)
	ep.self = rep
	ep.timeout = time.Duration(to) * time.Millisecond
	ep.array = make([]*InstList, nrep)
	ep.rpc = make([]chan interface{}, nrep)
	for i := int64(0); i < nrep; i++ {
		fileName := dir + strconv.FormatInt(i, 10) + ".dat"
		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_SYNC, 0644)
		if err != nil {
			log.Print(err)
			return nil
		}
		stat, err := file.Stat()
		if err != nil {
			log.Print(err)
			return nil
		}
		lst := &InstList{
			Mu:      sync.RWMutex{},
			LogFile: file,
			Offset:  common.InstanceID(int64(stat.Size()) / common.COMMAND_SIZE),
			Pending: make([]*StatefulInst, 0),
		}
		ep.array[i] = lst
		ep.rpc[i] = make(chan interface{}, buff)
	}
	endpoint := common.GetEnv("EPAXOS_LISTEN", "0.0.0.0:23330")
	addr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		log.Println(err)
		return nil
	}
	log.Printf("ListenUDP on %s\n", endpoint)
	ep.udp, err = net.ListenUDP("udp", addr)
	ep.udp.SetReadBuffer(int(rbuff))
	ep.udp.SetWriteBuffer(int(wbuff))
	if err != nil {
		log.Println(err)
		return nil
	}
	ep.probes = make(map[int64]chan bool)

	ep.data = make(map[common.Key]common.Value)
	ep.peers = nrep

	ep.sem = semaphore.NewWeighted(int64(pipe))
	ep.ctx = context.TODO()
	ep.isms = make(map[common.InstanceID]*InstStateMachine)
    ep.psmBook = make(map[common.InstRef]*PrepareStateMachine)

	ep.nextInst = ep.array[ep.self].Offset
	return ep
}

type logWriter struct {
	Id common.ReplicaID
}

func (writer *logWriter) Write(bytes []byte) (int, error) {
	return fmt.Printf(
		"%s #%02d %s",
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

	if rep < 0 || rep >= nrep {
		log.Fatalf("Invalid rep %d out of nrep %d", rep, nrep)
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
