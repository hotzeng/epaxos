package main

import (
	"epaxos/common"
	"errors"
	"fmt"
	"github.com/docopt/docopt-go"
	"log"
	"net"
	"sync"
	"time"
)

var VERSION string

var configEPaxos struct {
	ServerFmt  string
	ServerBias int64
	Buffer     int64
	NReps      int64
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
	return fmt.Printf(
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
    -f <fmt>     server endpoints format string
	-B <bias>    server format bias
	-b <buffer>  size of write buffer
	-n <nreps>   number of servers
`
	parser := &docopt.Parser{OptionsFirst: true}
	args, _ := parser.ParseArgs(usage, nil, "epaxos-client version "+VERSION)

	var err error
	configEPaxos.ServerFmt, err = args.String("-f")
	if err != nil {
		log.Fatal(err)
	}
	if args["-B"] == nil {
		configEPaxos.ServerBias = 0
	} else {
		bias, err := args.Int("-B")
		if err != nil {
			log.Fatal(err)
		}
		configEPaxos.ServerBias = int64(bias)
	}
	if args["-b"] == nil {
		configEPaxos.Buffer = 1024
	} else {
		buff, err := args.Int("-b")
		if err != nil {
			log.Fatal(err)
		}
		configEPaxos.Buffer = int64(buff)
	}
	nreps, err := args.Int("-n")
	if err != nil {
		log.Fatal(err)
	}
	configEPaxos.NReps = int64(nreps)

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
		return cmdProbe(ep, argv)
	}
	return errors.New("Command not found")
}

func cmdProbe(ep *EPaxosCluster, argv []string) error {
	usage := `usage: client probe [-v]
options:
	-h, --help
	-v, --verbose        be verbose
`
	args, _ := docopt.ParseArgs(usage, argv, "epaxos-client version "+VERSION)

	verbose, err := args.Bool("--verbose")
	if err != nil {
		return err
	}

	return ep.probeAll(verbose)
}
