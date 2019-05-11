package main

import (
	"epaxos/common"
	"errors"
	"fmt"
	"github.com/docopt/docopt-go"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var VERSION string

var configEPaxos struct {
	Verbose    bool
	ServerFmt  string
	ServerBias int64
	Buffer     int64
	WBuffer    int
	RBuffer    int
	NReps      int64
	TimeOut    time.Duration
}

type EPaxosCluster struct {
	mplxL   sync.Mutex
	mplx    map[int64]chan bool
	udp     []*net.UDPConn
	rpc     []chan interface{}
	inbound []chan interface{}
}

func NewEPaxosCluster() *EPaxosCluster {
	nrep := configEPaxos.NReps
	ep := new(EPaxosCluster)
	ep.udp = make([]*net.UDPConn, nrep)
	ep.rpc = make([]chan interface{}, nrep)
	ep.inbound = make([]chan interface{}, nrep)
	for i := int64(0); i < nrep; i++ {
		ep.rpc[i] = make(chan interface{}, configEPaxos.Buffer)
		ep.inbound[i] = make(chan interface{}, configEPaxos.Buffer)
	}
	ep.mplx = make(map[int64]chan bool)
	return ep
}

type logWriter struct {
}

func (*logWriter) Write(bytes []byte) (int, error) {
	return fmt.Fprintf(
		os.Stderr,
		"%s %s",
		time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		string(bytes),
	)
}

func main() {
	logW := new(logWriter)
	log.SetFlags(log.Lshortfile)
	log.SetOutput(logW)
	common.InitializeRand()

	usage := `usage: client [options] <command> [<args>...]
options:
	-v, --verbose   trace every package
	-f <fmt>        server endpoints [default: localhost:2333%d]
	-B <bias>       server format bias [default: 0]
	-q <buffer>     size of write queue [default: 1024]
	--write <wb>    size of udp write buffer [default: 1048576]
	--read <rb>     size of udp read buffer [default: 1048576]
	-n <nreps>      number of servers [default: 1]
	-t <timeout>    seconds to wait [default: 5.0]
`
	parser := &docopt.Parser{OptionsFirst: true}
	args, _ := parser.ParseArgs(usage, nil, "epaxos-client version "+VERSION)

	var err error
	configEPaxos.Verbose, err = args.Bool("--verbose")
	if err != nil {
		log.Fatal(err)
	}

	configEPaxos.ServerFmt, err = args.String("-f")
	if err != nil {
		log.Fatal(err)
	}

	bias, err := args.Int("-B")
	if err != nil {
		log.Fatal(err)
	}
	configEPaxos.ServerBias = int64(bias)

	buff, err := args.Int("-q")
	if err != nil {
		log.Fatal(err)
	}
	configEPaxos.Buffer = int64(buff)

	wbuff, err := args.Int("--write")
	if err != nil {
		log.Fatal(err)
	}
	configEPaxos.WBuffer = wbuff

	rbuff, err := args.Int("--read")
	if err != nil {
		log.Fatal(err)
	}
	configEPaxos.RBuffer = rbuff

	nreps, err := args.Int("-n")
	if err != nil {
		log.Fatal(err)
	}
	configEPaxos.NReps = int64(nreps)

	to, err := args.Float64("-t")
	if err != nil {
		log.Fatal(err)
	}
	configEPaxos.TimeOut = time.Duration(1000*to) * time.Millisecond

	ep := NewEPaxosCluster()
	err = ep.forkUdp()
	if err != nil {
		log.Fatal(err)
	}

	cmd := args["<command>"].(string)
	cmdArgs := args["<args>"].([]string)

	err = runCommand(ep, cmd, cmdArgs)
	if err != nil {
		log.Fatal(err)
	}
}

func runCommand(ep *EPaxosCluster, cmd string, args []string) error {
	argv := append([]string{cmd}, args...)
	switch cmd {
	case "probe":
		return ep.cmdProbe(argv)
	case "put":
		return ep.cmdPut(argv)
	case "put-get", "get":
		return ep.cmdPutGet(argv)
	case "batch-put":
		return ep.cmdBatchPut(argv)
	}
	return errors.New("Command not found")
}
