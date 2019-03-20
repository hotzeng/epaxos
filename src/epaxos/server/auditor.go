package main

import (
	"epaxos/common"
	"log"
)

func (ep *EPaxos) putAt(ref common.InstRef, obj *StatefulInst) {
	my := ep.array[ref.Replica]
	id := int(ref.Inst)
	my.Mu.Lock()
	defer my.Mu.Unlock()
	for int(my.Offset)+len(my.Pending) < id-1 {
		if ep.verbose {
			log.Printf("Out-of-order append happen on %d, from %d to %d", ref.Replica, int(my.Offset)+len(my.Pending), id)
		}
		my.Pending = append(my.Pending, nil)
	}
	if int(my.Offset)+len(my.Pending) <= id {
		if ep.verbose {
			log.Printf("Appending to %d, offset %d, from %d to %d", ref.Replica, my.Offset, len(my.Pending), id)
		}
		my.Pending = append(my.Pending, obj)
	} else {
		if ep.verbose {
			log.Printf("Rewriting to %d, offset %d, from %d to %d", ref.Replica, my.Offset, len(my.Pending), id)
		}
		my.Pending[ref.Inst-my.Offset] = obj
	}
}

func (ep *EPaxos) ProcessPreAccept(req common.PreAcceptMsg) {
	if ep.verbose {
		log.Printf("Auditor %d received PreAcceptMsg!", ep.self)
	}
	interf := make([]common.InstRef, 0)
	seqMax := req.Inst.Seq
	for index1, oneReplica := range ep.array {
		func() {
			oneReplica.Mu.RLock()
			defer oneReplica.Mu.RUnlock()
			for index2, oneInst := range oneReplica.Pending {
				if oneInst == nil {
					continue
				}
				if interfCmd(oneInst.inst.Cmd, req.Inst.Cmd) {
					interf = append(interf, common.InstRef{
						Replica: common.ReplicaID(index1),
						Inst:    common.InstanceID(index2),
					})
					if oneInst.inst.Seq+1 > seqMax {
						seqMax = oneInst.inst.Seq + 1
					}
				}
			}
		}()
	}
	req.Inst.Seq = seqMax
	compareMerge(&req.Inst.Deps, interf)
	obj := &StatefulInst{
		inst:  req.Inst,
		state: PreAccepting,
	}

	// check if need to fill in null elements in the Pending slice
	ep.putAt(req.Id, obj)

	// prepare PreAcceptOK msg and reply
	ep.rpc[req.Id.Replica] <- common.PreAcceptOKMsg{
		Id:     req.Id,
		Inst:   req.Inst,
		Sender: ep.self,
	}
	if ep.verbose {
		log.Printf("Auditor %d replied PreAcceptOKMsg!", ep.self)
	}
}

func (ep *EPaxos) ProcessAccept(req common.AcceptMsg) {
	obj := &StatefulInst{
		inst:  req.Inst,
		state: Accepting,
	}
	ep.putAt(req.Id, obj)
	ep.rpc[req.Id.Replica] <- common.AcceptOKMsg{
		Id:     req.Id,
		Inst:   req.Inst,
		Sender: ep.self,
	}
}

func (ep *EPaxos) ProcessCommit(req common.CommitMsg) {
	obj := &StatefulInst{
		inst:  req.Inst,
		state: Committing,
	}
	ep.putAt(req.Id, obj)
}

func (ep *EPaxos) ProcessPrepare(req common.PrepareMsg) {
	// TODO
}

func (ep *EPaxos) ProcessTryPreAccept(req common.TryPreAcceptMsg) {
	// TODO
}

func (ep *EPaxos) ProcessTryPreAcceptOK(req common.TryPreAcceptOKMsg) {
	// TODO
}
