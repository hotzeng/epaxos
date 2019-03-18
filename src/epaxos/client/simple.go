package main

import (
	"epaxos/common"
	"errors"
	"log"
	"math/rand"
	"time"
)

func (ep *EPaxosCluster) doPut(verbose bool, server common.ReplicaID, key common.Key, val common.Value) error {
	if verbose {
		log.Printf("PUT: %d=%d to %d start", key, val, server)
	}
	rnd := rand.Int63()
	ep.rpc[server] <- common.RequestMsg{
		MId: rnd,
		Cmd: common.Command{
			Cmd:   common.CmdPut,
			Key:   key,
			Value: val,
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
		log.Printf("PUT: %d=%d to %d committed", key, val, server)
	}
	return nil
}
