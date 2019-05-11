package main

import (
	"epaxos/common"
	"log"
)

func (ep *EPaxos) putAt(ref common.InstRef, obj *StatefulInst) {
	my := ep.array[ref.Replica]
	id := int(ref.Inst)
	if ep.verbose {
		log.Printf("putting to #%02d.%d: %+v", ref.Replica, ref.Inst, obj)
	}
	my.Mu.Lock()
	defer my.Mu.Unlock()
	if ref.Inst < my.Offset {
		log.Printf("Warning: old committeed messsage duplicated: %d.%d=%+v", ref.Replica, ref.Inst, obj)
		return
	}
	for {
		ca := int(my.Offset) + len(my.Pending)
		if ca < id {
			if ep.verbose {
				log.Printf("Out-of-order append happen on %d, from %d to %d: nil", ref.Replica, ca, id)
			}
			my.Pending = append(my.Pending, nil)
		} else if ca == id {
			if ep.verbose {
				log.Printf("Appending to %d, offset %d, from %d to %d: %v", ref.Replica, my.Offset, ca, id, obj)
			}
			my.Pending = append(my.Pending, obj)
			break
		} else {
			if ep.verbose {
				log.Printf("Rewriting to %d, offset %d, from %d to %d: %v", ref.Replica, my.Offset, ca, id, obj)
			}
			my.Pending[ref.Inst-my.Offset] = obj
			break
		}
	}
	if ep.verbose {
		log.Printf("done put to #%02d.%d: %+v", ref.Replica, ref.Inst, my.Pending[ref.Inst-my.Offset])
	}
}

func (ep *EPaxos) ProcessPreAccept(req common.PreAcceptMsg) {
	if ep.verbose {
		log.Printf("Auditor #%02d.%d received PreAcceptOKMsg!", req.Id.Replica, req.Id.Inst)
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
		inst:   req.Inst,
		state:  PreAccepting,
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
		log.Printf("Auditor #%02d.%d replied PreAcceptOKMsg!", req.Id.Replica, req.Id.Inst)
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
	if ep.verbose {
		log.Printf("Auditor #%02d.%d replied AcceptOKMsg!", req.Id.Replica, req.Id.Inst)
	}
}

func (ep *EPaxos) ProcessCommit(req common.CommitMsg) {
	obj := &StatefulInst{
		inst:  req.Inst,
		state: Finished,
	}
	ep.putAt(req.Id, obj)
	ep.mustAppendLogs(req.Id.Replica)
	if ep.verbose {
		log.Printf("Auditor #%02d.%d committed!", req.Id.Replica, req.Id.Inst)
	}
}

func (ep *EPaxos) ProcessPrepare(req common.PrepareMsg) {
	// TODO
    // increase ballot number
    ep.rpc[req.Id.Replica] <- common.PrepareOKMsg{
        Ack:    true,
        Ballot: ep.array[req.Id.Replica].Pending[req.Id.Inst].inst.Ballot,
        Id:     req.Id,
        Sender: ep.self,
        State:  ep.array[req.Id.Replica].Pending[req.Id.Inst].state}
    if ep.verbose {
        log.Printf("Auditor #%02.%d replied PrepareOKMsg!", req.Id.Replica, req.Id.Inst)
    }
}

func (ep *EPaxos) ProcessTryPreAccept(req common.TryPreAcceptMsg) {
	// TODO
}

func (ep *EPaxos) ProcessTryPreAcceptOK(req common.TryPreAcceptOKMsg) {
	// TODO
}
