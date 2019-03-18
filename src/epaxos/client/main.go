package main

import "log"
import "fmt"
import "time"
import "errors"
import "github.com/docopt/docopt-go"

var VERSION string

var configEPaxos struct {
	ServerFmt string
	NReps     int64
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

	usage := `usage: client [options] <command> [<args>...]
options:
    -f <fmt>     server endpoints format string
	-n <nreps>   number of servers
`
	parser := &docopt.Parser{OptionsFirst: true}
	args, _ := parser.ParseArgs(usage, nil, "epaxos-client version "+VERSION)

	var err error
	configEPaxos.ServerFmt, err = args.String("-f")
	if err != nil {
		log.Fatal(err)
	}
	nreps, err := args.Int("-n")
	if err != nil {
		log.Fatal(err)
	}
	configEPaxos.NReps = int64(nreps)

	cmd := args["<command>"].(string)
	cmdArgs := args["<args>"].([]string)

	err = runCommand(cmd, cmdArgs)
	if err != nil {
		log.Fatal(err)
	}
}

func runCommand(cmd string, args []string) error {
	argv := append([]string{cmd}, args...)
	switch cmd {
	case "probe":
		return cmdProbe(argv)
	}
	return errors.New("Command not found")
}

func cmdProbe(argv []string) error {
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

	return probeAll(verbose)
}
