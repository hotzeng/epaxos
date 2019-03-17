package main

import "os"
import "sync"
import "log"
import "strconv"
import "net"
import "net/rpc"
import "epaxos/common"

type InstState int32
type LeaderState int32

const (
	PreAccepted InstState = 0
	Accepted    InstState = 1
	Committed   InstState = 2
	Prepare     InstState = 3
)

// by yuzeng
// the states for command leader
const (
    LeaderPreAccept     LeaderState = 0
    LeaderPreAcceptOK   LeaderState = 1
    LeaderAccept        LeaderState = 2
    LeaderCommit        LeaderState = 3
    LeaderIdle          LeaderState = 4
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

    // by yuzeng
    state   LeaderState
    // channels for state transitions
    getReq          chan bool
    getPreAcceptOK  chan bool
    selectFastPath  chan bool
    getAcceptOK     chan bool
}

func NewEPaxos(nrep int64, rep common.ReplicaID) *EPaxos {
	dir := common.GetEnv("EPAXOS_DATA_PREFIX", "./data-")
	ep := new(EPaxos)
	ep.self = rep
	ep.array = make([]*InstList, nrep)
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
	}
	err := ep.recoverFromLog()
	if err != nil {
		log.Print(err)
		return nil
	}
	return ep
}

func (*EPaxos) HelloWorld(name string, ret *string) error {
	log.Println(name)
	*ret = "Hello " + name
	return nil
}

func main() {
	endpoint := common.GetEnv("EPAXOS_LISTEN", "0.0.0.0:23333")
	nrep, err := strconv.ParseInt(common.GetEnv("EPAXOS_NREPLICAS", "1"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	rep, err := strconv.ParseInt(common.GetEnv("EPAXOS_REPLICA_ID", "0"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	addy, err := net.ResolveTCPAddr("tcp", endpoint)
	if err != nil {
		log.Fatal(err)
	}

	inbound, err := net.ListenTCP("tcp", addy)
	if err != nil {
		log.Fatal(err)
	}

	ep := NewEPaxos(nrep, common.ReplicaID(rep))
	if ep == nil {
		log.Fatal("EPaxos creation failed")
	}
	rpc.Register(ep)
	rpc.Accept(inbound)
}
