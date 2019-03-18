package main

import (
	"epaxos/common"
	"log"
	"math/rand"
	"sync"
)

func (ep *EPaxosCluster) probeAll(verbose bool) error {
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
	}
	wg.Wait()
	if verbose {
		log.Print("Done probeAll")
	}
	return nil
}

func (ep *EPaxosCluster) probeOne(verbose bool, id common.ReplicaID) error {
	if verbose {
		log.Printf("Start probeOne %d", id)
	}

	rnd := rand.Int63()
	ep.rpc[id] <- common.KeepMsg{MId: rnd}

	for {
		msg := <-ep.inbound[id]
		m, ok := msg.(common.KeepMsg)
		if ok && m.MId == rnd {
			break
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
	}
	for {
		msg := <-ep.inbound[id]
		m, ok := msg.(common.ProbeReqMsg)
		if ok {
			if _, ok := msgs[m.MId]; ok {
				delete(msgs, m.MId)
				if len(msgs) == 0 {
					break
				}
			}
		}
	}
	if verbose {
		log.Printf("Done probeOne %d", id)
	}
	return nil
}
