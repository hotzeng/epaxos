package main

import (
	"epaxos/common"
	"log"
	"time"
)

// EPaxos only distributes msgs from client and to Instance
// state machines. However, the state machines will send messages
// to other replicas directly, without going through the EPaxos

// Struct for instance state machine
type InstStateMachine struct {
	instId common.InstanceID // which instID it is processing
	obj    *StatefulInst

	// PreAccept phase
	paChan chan common.PreAcceptOKMsg
	paLeft int64                     // how many left
	paBook map[common.ReplicaID]bool // which received
	paFast bool

	// Accept phase
	acChan chan common.AcceptOKMsg
	acLeft int64                     // how many left
	acBook map[common.ReplicaID]bool // which received

	// Commit phase
	cmChan chan bool
}

func NewInstStateMachine() *InstStateMachine {
	ism := new(InstStateMachine)
	return ism
}

// func to deal with client request
func (ep *EPaxos) RpcRequest(req common.RequestMsg, res *common.RequestOKMsg) error {
	err := ep.sem.Acquire(ep.ctx, 1)
	if err != nil {
		return err
	}
	defer ep.sem.Release(1)

	ism := NewInstStateMachine()
	ism.instId = func() common.InstanceID {
		ep.ismsL.Lock()
		defer ep.ismsL.Unlock()
		instId := ep.nextInst
		ep.nextInst++
		ep.isms[instId] = ism
		return instId
	}()
	defer func() {
		ep.ismsL.Lock()
		defer ep.ismsL.Unlock()
		delete(ep.isms, ism.instId)
	}()

	if ep.verbose {
		log.Printf("Leader started processing request %d", ism.instId)
	}

	return ep.runISM(ism, req.Cmd)
}

func (ep *EPaxos) ProcessPreAcceptOK(req common.PreAcceptOKMsg) {
	instId := req.Id.Inst
	ism := func() *InstStateMachine {
		ep.ismsL.RLock()
		defer ep.ismsL.RUnlock()
		return ep.isms[instId]
	}()
	if ism.paChan != nil {
		ism.paChan <- req
	} else {
		log.Printf("Warning: attempt to write to nil paChan at %d", instId)
	}
}

func (ep *EPaxos) ProcessAcceptOK(req common.AcceptOKMsg) {
	instId := req.Id.Inst
	ism := func() *InstStateMachine {
		ep.ismsL.RLock()
		defer ep.ismsL.RUnlock()
		return ep.isms[instId]
	}()
	if ism.acChan != nil {
		ism.acChan <- req
	} else {
		log.Printf("Warning: attempt to write to nil acChan at %d", instId)
	}
}

func (ep *EPaxos) ProcessPrepareOK(req common.PrepareOKMsg) {
	// TODO
}

// help func to check the interference between two commands
func interfCmd(cmd1 common.Command, cmd2 common.Command) bool {
	return cmd1.Key == cmd2.Key &&
		((cmd1.CmdType == common.CmdPut && cmd2.CmdType == common.CmdNoOp) ||
			(cmd1.CmdType == common.CmdNoOp && cmd2.CmdType == common.CmdPut) ||
			(cmd1.CmdType == common.CmdPut && cmd2.CmdType == common.CmdPut &&
				cmd1.Value != cmd2.Value))
}

// help func to compare and merge two dep sets
// TODO: make a more fast implementation
func compareMerge(dep1 *[]common.InstRef, dep2 []common.InstRef) bool { // return true if the same
	ret := true
	for _, id2 := range dep2 {
		exist := false
		for _, id1 := range *dep1 {
			if id1 == id2 {
				exist = true
				break
			}
		}
		if !exist {
			ret = false
			*dep1 = append(*dep1, id2)
		}
	}
	if !ret {
		return ret
	}
	if len(*dep1) > len(dep2) {
		return false
	}
	return ret
}

// start Instance state machine after receiving a request from client
func (ep *EPaxos) runISM(ism *InstStateMachine, cmd common.Command) error {
	if ep.verbose {
		log.Printf("Leader %d in Start Phase", ism.instId)
	}
	id := common.InstRef{Replica: ep.self, Inst: ism.instId}
	deps := make([]common.InstRef, 0)
	seqMax := common.Sequence(0)
	for index1, oneReplica := range ep.array {
		func() {
			oneReplica.Mu.RLock()
			defer oneReplica.Mu.RUnlock()
			for index2, oneInst := range oneReplica.Pending {
				if oneInst == nil {
					continue
				}
				if interfCmd(oneInst.inst.Cmd, cmd) {
					deps = append(deps, common.InstRef{Replica: common.ReplicaID(index1), Inst: common.InstanceID(index2)})
					if oneInst.inst.Seq > seqMax {
						seqMax = oneInst.inst.Seq
					}
				}
			}
		}()
	}
	seq := seqMax + 1
	ism.obj = &StatefulInst{
		inst: common.Instance{
			Cmd:  cmd,
			Seq:  seq,
			Deps: deps,
		},
	}
	ep.putAt(id, ism.obj)

	// send PreAccept to all other replicas in F
	// F := ep.peers / 2
	F := ep.peers
	ism.obj.state = PreAccepting
	ism.paChan = make(chan common.PreAcceptOKMsg)
	ism.paBook = make(map[common.ReplicaID]bool)
	ism.paLeft = F - int64(1)
	ism.paFast = true
	ism.obj.state = PreAccepting
	ep.makeMulticast(common.PreAcceptMsg{
		Id:   id,
		Inst: ism.obj.inst,
	}, ism.paLeft)
	if ep.verbose {
		log.Printf("Leader %d finished MultiCast for PreAccept", ism.instId)
	}

	for {
		switch ism.obj.state {
		case PreAccepting:
			if ep.verbose {
				log.Printf("Leader %d in PreAccepting Phase", ism.instId)
			}
			select {
			case msg := <-ism.paChan:
				if ep.verbose {
					log.Printf("Leader %d in PreAccepting Phase %d left", ism.instId, ism.paLeft)
				}
				// if the msg has been received from the sender, break
				if _, ok := ism.paBook[msg.Sender]; ok {
					log.Printf("Warning: duplicated msg from %d, ignored", msg.Sender)
					break
				}
				ism.paBook[msg.Sender] = true
				ism.paLeft--
				// compare seq and dep
				if msg.Inst.Seq > ism.obj.inst.Seq {
					ism.paFast = false
					ism.obj.inst.Seq = msg.Inst.Seq
				}
				if !compareMerge(&ism.obj.inst.Deps, msg.Inst.Deps) {
					ism.paFast = false
				}
				if ep.verbose {
					if ism.paFast {
						log.Printf("Leader %d needs %d more PreAcceptMsg %v", ism.instId, ism.paLeft, ism.obj.inst)
					} else {
						log.Printf("Leader %d (no-fast) needs %d more PreAcceptMsg %v", ism.instId, ism.paLeft, ism.obj.inst)
					}
				}
				if ism.paLeft > 0 {
					break
				}
				ism.paChan = nil // Prevent future dup msg
				ism.paBook = nil
				if ism.paFast {
					ism.obj.state = Committing
					if ep.verbose {
						log.Printf("Leader %d chose Fast Path", ism.instId)
					}
					break
				}
				ism.obj.state = Accepting
				ism.acChan = make(chan common.AcceptOKMsg)
				ism.acBook = make(map[common.ReplicaID]bool)
				ism.acLeft = ep.peers / 2
				// send accepted msg to replicas
				ep.makeMulticast(common.AcceptMsg{
					Id:   id,
					Inst: ism.obj.inst,
				}, ism.acLeft)
				if ep.verbose {
					log.Printf("Leader %d finished MultiCase for Accept", ism.instId)
				}
			case <-time.After(ep.timeout):
				// If not received enough OK msg, re-multicast
				if ep.verbose {
					log.Printf("Time out! Leader %d re-send MultiCast for PreAccept Msg!", ism.instId)
				}
				ism.paLeft = F - int64(1)
				ism.paBook = make(map[common.ReplicaID]bool)
				ep.makeMulticast(common.PreAcceptMsg{
					Id:   id,
					Inst: ism.obj.inst,
				}, ism.paLeft)
			}

		case Accepting:
			if ep.verbose {
				log.Printf("Leader %d in Accepting Phase!", ism.instId)
			}
			select {
			case msg := <-ism.acChan:
				// if the msg has been received from the sender, break
				if _, ok := ism.acBook[msg.Sender]; ok {
					log.Printf("Warning: duplicated msg from %d, ignored", msg.Sender)
					break
				}
				ism.acBook[msg.Sender] = true
				ism.acLeft--
				if ism.acLeft > 0 {
					break
				}
				ism.acChan = nil // Prevent future dup msg
				ism.acBook = nil
				ism.obj.state = Committing
				if ep.verbose {
					log.Printf("Leader %d received enough AcceptMsg!", ism.instId)
				}
			case <-time.After(ep.timeout):
				// If not received enough OK msg, re-multicast
				if ep.verbose {
					log.Printf("Time out! Leader %d re-send MultiCast for Accept Msg!", ism.instId)
				}
				ism.acLeft = ep.peers / 2
				ism.acBook = make(map[common.ReplicaID]bool)
				ep.makeMulticast(common.AcceptMsg{
					Id:   id,
					Inst: ism.obj.inst,
				}, ism.acLeft)
			}

		case Committing:
			if ep.verbose {
				log.Printf("Leader %d in Committing Phase!", ism.instId)
			}
			ism.obj.state = Finished
			ism.cmChan = make(chan bool)
			go ep.mustAppendLogs(ep.self)
			// wait for persiter to finish
			<-ism.cmChan
			if ep.verbose {
				log.Printf("Leader %d persisted, broadcast and return to client", ism.instId)
			}
			// send commit msg to replicas
			ep.makeMulticast(common.CommitMsg{
				Id:   id,
				Inst: ism.obj.inst,
			}, ep.peers-int64(1))
			return nil
		}
	}
}

func (ep *EPaxos) ProcessRequest(req common.RequestMsg) common.RequestOKMsg {
	var res common.RequestOKMsg
	res.MId = req.MId
	err := ep.RpcRequest(req, &res)
	if err != nil {
		log.Println(err)
		res.Err = true
	}
	return res
}

func (ep *EPaxos) ProcessRequestAndRead(req common.RequestAndReadMsg) common.RequestAndReadOKMsg {
	// TODO
	return common.RequestAndReadOKMsg{MId: req.MId}
}
