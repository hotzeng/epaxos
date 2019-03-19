package main

import (
	"epaxos/common"
	"errors"
	"log"
	"math"
)

// EPaxos only distributes msgs from client and to Instance
// state machines. However, the state machines will send messages
// to other replicas directly, without going through the EPaxos

// Struct for instance state machine
type instStateMachine struct {
	self      common.InstanceID // which instID it is processing
	state     InstState
	innerChan chan interface{} // the chan used to communicate with replica

	// preAccept phase
	FQuorum       []common.ReplicaID        // the quorun to send preaccept
	preAcceptNo   int                       // number for received preAccept msg
	preAcceptBook map[common.ReplicaID]bool // record from which preAccept is received

	// preAcceptOK phase
	chooseFast bool
	seqOK      common.Sequence  // the max seq
	depOK      []common.InstRef // the union of all deps

	// accept phase
	acceptNo int // the number of received accept msg
}

// func to deal with client request
func (ep *EPaxos) RpcRequest(req common.RequestMsg, res *common.RequestOKMsg) error {
	if ep.chanHead.pointer == ep.chanTail.pointer+1 {
		return errors.New("No free channel for new instance machine")
	}
	ep.lastInst.mu.Lock()
	ep.lastInst.InstanceID++
	localInst := ep.lastInst.InstanceID
	ep.lastInst.mu.Unlock()

	ep.chanTail.mu.Lock()
	ep.chanTail.pointer++
	localChan := ep.chanTail.pointer
	ep.chanTail.mu.Unlock()

	ep.inst2Chan[localInst] = localChan

	go ep.startInstanceState(localInst, req.Cmd, ep.innerChan[localChan])

	for {
		reply, more := <-ep.innerChan[localChan]
		if r, ok := reply.(common.RequestOKMsg); ok && r.Err == false {
			// TODO: release the channel
			break
		} else if !more {
			return errors.New("Instance State Machine did not respond correctly")
		}
	}
	return nil
}

func (ep *EPaxos) ProcessPreAcceptOK(req common.PreAcceptOKMsg) error {
	instId := req.Id.Inst
	ep.innerChan[instId] <- req
	return nil
}

func (ep *EPaxos) ProcessAcceptOK(req common.AcceptOKMsg) error {
	instId := req.Id.Inst
	ep.innerChan[instId] <- req
	return nil
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
	for index2, id2 := range dep2 {
		exist := false
		for index1, id1 := range *dep1 {
			if id1 == id2 {
				exist = true
				break
			}
		}
		if exist == false {
			ret = false
			*dep1 = append(*dep1, id2)
		}
	}
	if ret == false {
		return ret
	}
	if len(*dep1) > len(dep2) {
		return false
	}
	return ret
}

// start Instance state machine after receiving a request from client
func (ep *EPaxos) startInstanceState(instId common.InstanceID, cmd common.Command, innerChan chan interface{}) {
	// ism abbreviated to instance state machine
	ism := &instStateMachine{}
	ism.self = instId
	ism.innerChan = innerChan
	ism.state = Start // starting from idle state, send preaccept to F
	ism.preAcceptNo = 0
	ism.chooseFast = true
	//var innerMsg interface{}

	for {
		switch ism.state {
		case Start:
			deps := make([]common.InstRef, 0)
			seqMax := common.Sequence(0)
			for index1, oneReplica := range ep.array {
				for index2, oneInst := range oneReplica.Pending {
					if interfCmd(oneInst.inst.Cmd, cmd) {
						deps = append(deps, common.InstRef{Replica: common.ReplicaID(index1), Inst: common.InstanceID(index2)})
						if oneInst.inst.Seq > seqMax {
							seqMax = oneInst.inst.Seq
						}
					}
				}
			}
			seq := seqMax + 1
			inst := &common.Instance{}
			inst.Cmd = cmd
			// modify and save instance atomically
			inst.Mu.Lock()
			inst.Seq = seq
			inst.Deps = deps
			// TODO: assign NDeps
			ep.array[ep.self].Pending[instId] = &StatefulInst{inst: *inst, state: PreAccepted}
			inst.Mu.Unlock()
			ism.seqOK = seq
			ism.depOK = deps

			// send PreAccept to all other replicas in F
			F := math.Floor(float64(ep.peers) / 2)
			sendMsg := &common.PreAcceptMsg{}
			sendMsg.Inst = *inst
			sendMsg.Id = common.InstRef{ep.self, instId}
			ism.FQuorum = ep.makeMulticast(sendMsg, int64(F-1))
			ism.preAcceptNo = int(F - 1)
			ism.state = PreAccepted

		case PreAccepted: // waiting for PreAccepOKMsg
			// wait for msg non-blockingly
			select {
			case innerMsg := <-ism.innerChan:
				if okMsg, ok := innerMsg.(common.PreAcceptOKMsg); ok == true {
					// check instId is correct
					if okMsg.Id.Inst != ism.self {
						// TODO: change this fatal command
						log.Fatal("Wrong inner msg!")
					}
					// if the msg has been received from the sender, break
					if ism.preAcceptBook[okMsg.Sender] == true {
						break
					}
					ism.preAcceptBook[okMsg.Sender] = true
					ism.preAcceptNo--
					// compare seq and dep
					if okMsg.Seq > ism.seqOK {
						ism.chooseFast = false
						ism.seqOK = okMsg.Seq
					}
					if compareMerge(&ism.depOK, okMsg.Deps) == false {
						ism.chooseFast = false
					}
					if ism.preAcceptNo == 0 {
						if ism.chooseFast == true {
							ism.state = Committed
						} else {
							ism.state = Accepted
							ism.acceptNo = int(math.Floor(float64(ep.peers) / 2))
							inst := &common.Instance{}
							inst.Cmd = cmd
							inst.Seq = ism.seqOK
							inst.Deps = ism.depOK
							ep.array[ep.self].Pending[instId] = &StatefulInst{inst: *inst, state: Accepted}
							// send accepted msg to replicas
							sendMsg := &common.AcceptMsg{}
							sendMsg.Id = common.InstRef{Replica: ep.self, Inst: instId}
							sendMsg.Inst = *inst
							ism.FQuorum = ep.makeMulticast(sendMsg, int64(math.Floor(float64(ep.peers)/2))) // TODO: change the number of Multicast for faster response
						}
					}
				}
			default:
			}

		case Accepted:
			select {
			case innerMsg := <-ism.innerChan:
				ism.acceptNo--
				if acceptMsg, ok := innerMsg.(common.AcceptOKMsg); ok != true {
					break
				}
				if ism.acceptNo == 0 {
					ism.state = Committed
				}
			}
		default:

		case Committed:
			inst := &common.Instance{}
			inst.Cmd = cmd
			inst.Seq = ism.seqOK
			inst.Deps = ism.depOK
			ep.array[ep.self].Pending[instId] = &StatefulInst{inst: *inst, state: Committed}
			// send commit msg to replicas
			sendMsg := &common.CommitMsg{}
			sendMsg.Id = common.InstRef{Replica: ep.self, Inst: instId}
			sendMsg.Inst = *inst
			ep.makeMulticast(sendMsg, ep.peers-1)
			ism.state = Idle
			close(innerChan)
			ism.innerChan <- common.RequestOKMsg{Err: false}
			return
		}
	}
}

func (ep *EPaxos) ProcessRequest(req common.RequestMsg) (common.RequestOKMsg, error) {
	var res common.RequestOKMsg
	res.MId = req.MId
	err := ep.RpcRequest(req, &res)
	return res, err
}

func (ep *EPaxos) ProcessRequestAndRead(req common.RequestAndReadMsg) (common.RequestAndReadOKMsg, error) {
	// TODO
	return common.RequestAndReadOKMsg{MId: req.MId}, nil
}
