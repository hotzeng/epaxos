package main

import (
	"epaxos/common"
	"errors"
	"github.com/docopt/docopt-go"
	"log"
	"math/rand"
	"time"
)

func (ep *EPaxosCluster) cmdPut(argv []string) error {
	usage := `usage: client put [-v] <server> <key> <value>
options:
	-h, --help
	-v, --verbose        be verbose
`
	args, _ := docopt.ParseArgs(usage, argv, "epaxos-client version "+VERSION)

	verbose, err := args.Bool("--verbose")
	if err != nil {
		return err
	}

	s, err := args.Int("<server>")
	if err != nil {
		return err
	}
	server := common.ReplicaID(s)

	k, err := args.Int("<key>")
	if err != nil {
		return err
	}
	key := common.Key(k)

	v, err := args.Int("<value>")
	if err != nil {
		return err
	}
	val := common.Value(v)

	if verbose {
		log.Printf("PUT: [%d]=%d to %d start", key, val, server)
	}
	rnd := rand.Int63()
	ep.rpc[server] <- common.RequestMsg{
		MId: rnd,
		Cmd: common.Command{
			CmdType: common.CmdPut,
			Key:     key,
			Value:   val,
		},
	}
	if verbose {
		log.Printf("PUT: RequestMsg %d sent to %d", rnd, server)
	}

loop:
	for {
		select {
		case msg := <-ep.inbound[server]:
			if m, ok := msg.(common.RequestOKMsg); ok && m.MId == rnd {
				if m.Err {
					log.Printf("Remote error during PUT %d", server)
					return errors.New("remote error")
				}
				break loop
			}
		case <-time.After(configEPaxos.TimeOut):
			log.Printf("PUT %d timeout \n", server)
			return errors.New("put timeout")
		}
	}

	if verbose {
		log.Printf("PUT: [%d]=%d to %d committed", key, val, server)
	}
	return nil
}

func (ep *EPaxosCluster) cmdPutGet(argv []string) error {
	usage := `usage: client get [-v] <server> <key>
       client put-get [-v] <server> <key> <value>
options:
	-h, --help
	-v, --verbose        be verbose
`
	args, _ := docopt.ParseArgs(usage, argv, "epaxos-client version "+VERSION)

	verbose, err := args.Bool("--verbose")
	if err != nil {
		return err
	}

	s, err := args.Int("<server>")
	if err != nil {
		return err
	}
	server := common.ReplicaID(s)

	k, err := args.Int("<key>")
	if err != nil {
		return err
	}
	key := common.Key(k)

	v, err := args.Int("<value>")
	cmd := common.CmdPut
	if args["<value>"] == nil {
		cmd = common.CmdNoOp
	} else {
		if err != nil {
			return err
		}
	}
	val := common.Value(v)

	if verbose {
		if cmd == common.CmdPut {
			log.Printf("PUT-GET: [%d]=%d to %d start", key, val, server)
		} else {
			log.Printf("GET: [%d] to %d start", key, server)
		}
	}
	rnd := rand.Int63()
	ep.rpc[server] <- common.RequestAndReadMsg{
		MId: rnd,
		Cmd: common.Command{
			CmdType: cmd,
			Key:     key,
			Value:   val,
		},
	}
	if verbose {
		if cmd == common.CmdPut {
			log.Printf("PUT-GET: RequestAndReadMsg %d sent to %d", rnd, server)
		} else {
			log.Printf("GET: RequestAndReadMsg %d sent to %d", rnd, server)
		}
	}

loop:
	for {
		select {
		case msg := <-ep.inbound[server]:
			if m, ok := msg.(common.RequestAndReadOKMsg); ok && m.MId == rnd {
				if m.Err {
					log.Printf("Remote error during [PUT-]GET %d", server)
					return errors.New("remote error")
				}
				if cmd == common.CmdPut {
					if m.Exist {
						log.Printf("Result: PUT-GET [%d]=%d == %d", key, val, m.Value)
					} else {
						log.Printf("Result: PUT-GET [%d]=%d == <nil>", key, val)
					}
				} else {
					if m.Exist {
						log.Printf("Result: GET [%d] == %d", key, m.Value)
					} else {
						log.Printf("Result: GET [%d] == <nil>", key)
					}
				}
				break loop
			}
		case <-time.After(configEPaxos.TimeOut):
			if cmd == common.CmdPut {
				log.Printf("PUT-GET %d timeout \n", server)
			} else {
				log.Printf("GET %d timeout \n", server)
			}
			return errors.New("put-get timeout")
		}
	}

	if verbose {
		if cmd == common.CmdPut {
			log.Printf("PUT-GET: [%d]=%d to %d committed", key, val, server)
		} else {
			log.Printf("GET: [%d] to %d committed", key, server)
		}
	}
	return nil
}
