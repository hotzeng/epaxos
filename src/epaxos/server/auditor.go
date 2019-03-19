package main

import (
	"epaxos/common"
	"fmt"
)

func (ep *EPaxos) ProcessPreAccept(req common.PreAcceptMsg) error {
	if ep.verbose {
		fmt.Printf("Auditor %d received PreAcceptMsg!\n", ep.self)
	}
	interf := make([]common.InstRef, 0)
	seqMax := req.Inst.Seq
	ep.mu.Lock()
	for index1, oneReplica := range ep.array {
		for index2, oneInst := range oneReplica.Pending {
			if interfCmd(oneInst.inst.Cmd, req.Inst.Cmd) {
				interf = append(interf, common.InstRef{Replica: common.ReplicaID(index1), Inst: common.InstanceID(index2)})
				if oneInst.inst.Seq+1 > seqMax {
					seqMax = oneInst.inst.Seq + 1
				}
			}
		}
	}
	// update deps
	compareMerge(&req.Inst.Deps, interf)
	inst := common.Instance{
		Cmd:  req.Inst.Cmd,
		Seq:  seqMax,
		Deps: req.Inst.Deps,
	}
	// check if need to fill in null elements in the Pending slice
	pendingLen := len(ep.array[req.Id.Replica].Pending)
	if pendingLen >= int(req.Id.Inst) {
		fmt.Println("Pending array is erroneously wrong")
	} else if pendingLen < int(req.Id.Inst)-1 {
		// if too short, add nil elements
		for i := pendingLen; i < int(req.Id.Inst)-1; i++ {
			ep.array[req.Id.Replica].Pending = append(ep.array[req.Id.Replica].Pending, &StatefulInst{})
		}
	}
	ep.array[req.Id.Replica].Pending = append(ep.array[req.Id.Replica].Pending, &StatefulInst{inst: inst, state: PreAccepted})
	ep.mu.Unlock()

	// prepare PreAcceptOK msg and reply
	sendMsg := common.PreAcceptOKMsg{
		Id:     common.InstRef{Replica: req.Id.Replica, Inst: req.Id.Inst},
		Inst:   inst,
		Sender: ep.self,
	}
	ep.rpc[req.Id.Replica] <- sendMsg
	if ep.verbose {
		fmt.Printf("Auditor %d replied PreAcceptOKMsg!\n", ep.self)
	}
	return nil
}

func (ep *EPaxos) ProcessAccept(req common.AcceptMsg) error {
	ep.mu.Lock()
	ep.array[req.Id.Replica].Pending[req.Id.Inst] = &StatefulInst{inst: req.Inst, state: Accepted}
	ep.mu.Unlock()
	ep.rpc[req.Id.Replica] <- common.AcceptOKMsg{Id: req.Id, Inst: req.Inst, Sender: ep.self}
	return nil
}

func (ep *EPaxos) ProcessCommit(req common.CommitMsg) error {
	ep.mu.Lock()
	ep.array[req.Id.Replica].Pending[req.Id.Inst] = &StatefulInst{inst: req.Inst, state: Committed}
	ep.mu.Unlock()
	return nil
}
