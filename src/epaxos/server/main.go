package main

import "os"
import "sync"
import "errors"
import "fmt"
import "log"
import "strconv"
import "net"
import "net/rpc"
import "epaxos/common"

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
	udp      *net.UDPConn
	rpc      []chan interface{}
}

func NewEPaxos(nrep int64, rep common.ReplicaID, endpoint string) *EPaxos {
	dir := common.GetEnv("EPAXOS_DATA_PREFIX", "./data-")
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
		ep.rpc[i] = make(chan interface{})
	}
	ep.inbound = &ep.rpc[ep.self]
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
	err = ep.recoverFromLog()
	if err != nil {
		log.Println(err)
		return nil
	}
	return ep
}

func (ep *EPaxos) ReadyProbe(payload string, ret *string) error {
	log.Printf("EPaxos.ReadyProbe with %s\n", payload)
	*ret = fmt.Sprintf("I'm EPaxos #%d", ep.self)
	return nil
}

func (ep *EPaxos) SendProbe(target common.ReplicaID, ret *string) error {
	log.Printf("EPaxos.SendProbe to %d\n", target)
	if int(target) >= len(ep.rpc) {
		return errors.New("out of range")
	}
	ep.rpc[target] <- common.ProbeMsg{}
	*ret = fmt.Sprintf("I'm EPaxos #%d", ep.self)
	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	endpoint := common.GetEnv("EPAXOS_LISTEN", "0.0.0.0:23333")
	nrep, err := strconv.ParseInt(common.GetEnv("EPAXOS_NREPLICAS", "1"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	rep, err := strconv.ParseInt(common.GetEnv("EPAXOS_REPLICA_ID", "0"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	addr, err := net.ResolveTCPAddr("tcp", endpoint)
	if err != nil {
		log.Fatal(err)
	}

	inbound, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	ep := NewEPaxos(nrep, common.ReplicaID(rep), endpoint)
	if ep == nil {
		log.Fatal("EPaxos creation failed")
	}

	err = ep.forkUdp()
	if err != nil {
		log.Fatal(err)
	}

	rpc.Register(ep)
	rpc.Accept(inbound)
}
