package main

import (
	"epaxos/common"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var VERSION string

type InstState int32

const (
	PreAccepted InstState = 0
	Accepted    InstState = 1
	Committed   InstState = 2
	Prepare     InstState = 3
)

type StatefulInst struct {
	inst  common.Instance
	state InstState
}

type InstList struct {
	Mu      sync.Mutex
	LogFile *os.File
	Offset  common.InstanceID
	Pending []*StatefulInst
}

type EPaxos struct {
	self     common.ReplicaID
	lastInst common.InstanceID
	array    []*InstList
	data     map[common.Key]common.Value
	inbound  *chan interface{}
	probesL  sync.Mutex
	probes   map[int64]chan bool
	udp      *net.UDPConn
	rpc      []chan interface{}
}

func NewEPaxos(nrep int64, rep common.ReplicaID) *EPaxos {
	dir := common.GetEnv("EPAXOS_DATA_PREFIX", "./data/data-")
	buff, err := strconv.ParseInt(common.GetEnv("EPAXOS_BUFFER", "1024"), 10, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	ep := new(EPaxos)
	ep.self = rep
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
	rand.Seed(time.Now().UTC().UnixNano() + rep)
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
