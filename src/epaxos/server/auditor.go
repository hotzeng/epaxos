package main

import "epaxos/common"

func (ep *EPaxos) ProcessPreAccept(req common.PreAcceptMsg) error {
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
	inst := &common.Instance{}
	inst.Cmd = req.Inst.Cmd
	inst.Seq = seqMax
	inst.Deps = req.Inst.Deps
	ep.array[req.Id.Replica].Pending[req.Id.Inst] = &StatefulInst{inst: *inst, state: PreAccepted}
	ep.mu.Unlock()

	// prepare PreAcceptOK msg and reply
	sendMsg := &common.PreAcceptOKMsg{}
	sendMsg.Id = common.InstRef{Replica: req.Id.Replica, Inst: req.Id.Inst}
	sendMsg.Inst = *inst
	sendMsg.Sender = ep.self
	ep.rpc[req.Id.Replica] <- sendMsg
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
