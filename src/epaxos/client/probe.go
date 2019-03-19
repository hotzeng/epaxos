package main

import (
	"epaxos/common"
	"errors"
	"github.com/docopt/docopt-go"
	"log"
	"math/rand"
	"sync"
	"time"
)

func (ep *EPaxosCluster) cmdProbe(argv []string) error {
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

	if verbose {
		log.Printf("Start probeAll for %d", len(ep.rpc))
	}
	var wg sync.WaitGroup
	wg.Add(len(ep.rpc))
	good := true
	for i := int64(0); i < configEPaxos.NReps; i++ {
		id := common.ReplicaID(i)
		go func() {
			defer wg.Done()
			err := ep.probeOne(verbose, id)
			if err != nil {
				log.Println(err)
				good = false
			}
		}()
		<-time.After(47 * time.Millisecond)
	}
	wg.Wait()
	if !good {
		return errors.New("remote errors")
	}
	if verbose {
		log.Println("Done probeAll")
	}
	return nil
}

func (ep *EPaxosCluster) probeOne(verbose bool, id common.ReplicaID) error {
	if verbose {
		log.Printf("Start probeOne %d", id)
	}

	rnd := rand.Int63()
	ep.rpc[id] <- common.KeepMsg{MId: rnd}

loop1:
	for {
		select {
		case msg := <-ep.inbound[id]:
			m, ok := msg.(common.KeepMsg)
			if ok && m.MId == rnd {
				break loop1
			}
		case <-time.After(configEPaxos.TimeOut):
			log.Printf("keep msg %d timeout \n", id)
			return errors.New("probe timeout")
		}
	}

	msgs := make(map[int64]bool)
	for i := int64(0); i < configEPaxos.NReps; i++ {
		id2 := common.ReplicaID(i)
		if id == id2 {
			continue
		}
		for {
			rnd = rand.Int63()
			if _, ok := msgs[rnd]; !ok {
				msgs[rnd] = true
				break
			}
		}
		ep.rpc[id] <- common.ProbeReqMsg{
			MId:     rnd,
			Replica: id2,
		}
		<-time.After(67 * time.Millisecond)
	}
	good := true
loop2:
	for {
		select {
		case msg := <-ep.inbound[id]:
			if m, ok := msg.(common.ProbeReqMsg); ok {
				if m.Replica == common.ReplicaID(-1) {
					log.Printf("Remote error during probeOne %d", id)
					good = false
				}
				if _, ok := msgs[m.MId]; ok {
					delete(msgs, m.MId)
					if len(msgs) == 0 {
						break loop2
					}
				}
			}
		case <-time.After(configEPaxos.TimeOut):
			log.Printf("probe req msg %d timeout \n", id)
			return errors.New("probe timeout")
		}
	}
	if !good {
		return errors.New("remote errors")
	}
	if verbose {
		log.Printf("Done probeOne %d", id)
	}
	return nil
}
