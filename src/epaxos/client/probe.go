package main

import (
	"context"
	"epaxos/common"
	"errors"
	"github.com/docopt/docopt-go"
	"golang.org/x/sync/semaphore"
	"log"
	"math/rand"
	"sync"
	"time"
)

func (ep *EPaxosCluster) cmdProbe(argv []string) error {
	usage := `usage: client probe [options]
options:
	-h, --help
	-v, --verbose        be verbose
	--focus              stress each node at a time
	--pipeline <pipe>    on-flight operations limit [default: 1024]
`
	args, _ := docopt.ParseArgs(usage, argv, "epaxos-client version "+VERSION)

	verbose, err := args.Bool("--verbose")
	if err != nil {
		return err
	}

	focus, err := args.Bool("--focus")
	if err != nil {
		return err
	}

	pipe, err := args.Int("--pipeline")
	if err != nil {
		return err
	}

	if verbose {
		log.Printf("Start probeAll for %d", len(ep.rpc))
	}
	sem := semaphore.NewWeighted(int64(pipe))
	ctx := context.TODO()
	var wg sync.WaitGroup
	wg.Add(len(ep.rpc))
	good := true
	for i := int64(0); i < configEPaxos.NReps; i++ {
		id := common.ReplicaID(i)
		go func() {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Println(err)
				good = false
				return
			}
			defer sem.Release(1)
			err := ep.probeOne(verbose, focus, id)
			if err != nil {
				log.Println(err)
				good = false
			}
		}()
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

func (ep *EPaxosCluster) probeOne(verbose, focus bool, id common.ReplicaID) error {
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

	if verbose {
		log.Printf("Mid probeOne %d", id)
	}

	msgs := make(map[int64]bool)
	for i := int64(0); i < configEPaxos.NReps; i++ {
		var id2 common.ReplicaID
		if focus {
			id2 = common.ReplicaID(i)
		} else {
			id2 = common.ReplicaID((int64(id) + i) % configEPaxos.NReps)
		}
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
	}
	good := true
	for {
		if len(msgs) == 0 {
			break
		}
		select {
		case msg := <-ep.inbound[id]:
			if m, ok := msg.(common.ProbeReqMsg); ok {
				if m.Replica == common.ReplicaID(-1) {
					log.Printf("Remote error during probeOne %d", id)
					good = false
				}
				if _, ok := msgs[m.MId]; ok {
					delete(msgs, m.MId)
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
