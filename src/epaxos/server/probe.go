package main

import (
	"epaxos/common"
	"errors"
	"log"
	"math/rand"
	"time"
)

func (ep *EPaxos) allocProbe() (int64, chan bool) {
	ep.probesL.Lock()
	defer ep.probesL.Unlock()
	var probeId int64
	for {
		probeId = rand.Int63()
		if _, ok := ep.probes[probeId]; !ok {
			break
		}
	}
	ch := make(chan bool)
	log.Printf("Probe: ProbeId %d allocated:", probeId)
	ep.probes[probeId] = ch
	return probeId, ch
}

func (ep *EPaxos) freeProbe(probeId int64) {
	ep.probesL.Lock()
	defer ep.probesL.Unlock()
	log.Printf("Probe: ProbeId %d freed:", probeId)
	delete(ep.probes, probeId)
}

func (ep *EPaxos) recvProbe(m *common.ProbeMsg) {
	if m.RequestReply {
		log.Printf("Probe: Received ProbeMsg from %d, will reply\n", m.Replica)
		ep.rpc[m.Replica] <- common.ProbeMsg{
			Replica:      ep.self,
			Payload:      m.Payload,
			RequestReply: false,
		}
	} else {
		log.Printf("Probe: Received ProbeMsg from %d\n", m.Replica)
		ep.probesL.Lock()
		defer ep.probesL.Unlock()
		log.Printf("Probe: ProbeId %d received:", m.Payload)
		if ch, ok := ep.probes[m.Payload]; ok {
			ch <- true
		}
	}
}

func (ep *EPaxos) sendProbe(target common.ReplicaID) error {
	log.Printf("Probe: EPaxos.sendProbe to %d start\n", target)
	if int(target) >= len(ep.rpc) {
		return errors.New("out of range")
	}
	if target == ep.self {
		return nil
	}
	probeId, ch := ep.allocProbe()
	defer ep.freeProbe(probeId)
	ep.rpc[target] <- common.ProbeMsg{
		Replica:      ep.self,
		Payload:      probeId,
		RequestReply: true,
	}
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		log.Printf("EPaxos.sendProbe to %d timeout \n", target)
		return errors.New("probe timeout")
	}
	log.Printf("Probe: EPaxos.sendProbe to %d succeed\n", target)
	return nil
}
