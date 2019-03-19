package main

import (
	"epaxos/common"
	"errors"
	"log"
	"math"
	"time"
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
	freeChanExist := false
	ep.mu.Lock()
	var freeChanNo ChannelID
	for i := ChannelID(0); i < CHAN_MAX; i++ {
		if bit, ok := ep.freeChan[i]; (!bit && ok) || !ok {
			freeChanExist = true
			freeChanNo = i
			break
		}
	}
	if !freeChanExist {
		return errors.New("No free channel for new instance machine")
	}
	ep.lastInst++
	localInst := ep.lastInst

	localChan := freeChanNo
	ep.inst2Chan[localInst] = localChan
	ep.freeChan[localChan] = true
	ep.mu.Unlock()

	if ep.verbose {
		log.Printf("Leader %d received request!", ep.self)
	}
	go ep.startInstanceState(localInst, req.Cmd, ep.innerChan[localChan])

	for {
		reply, more := <-ep.innerChan[localChan]
		if r, ok := reply.(common.RequestOKMsg); ok && !r.Err {
			res.Err = false
			ep.freeChan[localChan] = false
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
func (ep *EPaxos) startInstanceState(instId common.InstanceID, cmd common.Command, innerChan chan interface{}) {
	// ism abbreviated to instance state machine
	ism := &instStateMachine{}
	ism.self = instId
	ism.innerChan = innerChan
	ism.state = Start // starting from idle state, send preaccept to F
	ism.preAcceptNo = 0
	ism.chooseFast = true
	ism.preAcceptBook = make(map[common.ReplicaID]bool)
	var paMsg common.PreAcceptMsg
	var aMsg common.AcceptMsg

	//var innerMsg interface{}

	for {
		switch ism.state {
		case Start:
			if ep.verbose {
				log.Printf("Leader %d in Start Phase", ep.self)
			}
			deps := make([]common.InstRef, 0)
			seqMax := common.Sequence(0)
			ep.mu.Lock()
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
			// modify and save instance atomically
			inst := common.Instance{
				Cmd:  cmd,
				Seq:  seq,
				Deps: deps,
			}
			// TODO: assign NDeps
			//ep.array[ep.self].Pending[instId] = &StatefulInst{inst: *inst, state: PreAccepted}
			ep.array[ep.self].Pending = append(ep.array[ep.self].Pending, &StatefulInst{inst: inst, state: PreAccepted})
			if len(ep.array[ep.self].Pending) != int(instId) {
				log.Println("Length of Instructions not correct!")
			}
			ep.mu.Unlock()
			ism.seqOK = seq
			ism.depOK = deps

			// send PreAccept to all other replicas in F
			//F := math.Floor(float64(ep.peers) / 2)
			F := ep.peers
			paMsg.Inst = inst
			paMsg.Id = common.InstRef{Replica: ep.self, Inst: instId}
			ism.FQuorum = ep.makeMulticast(paMsg, int64(F-1))
			if ep.verbose {
				log.Printf("Leader %d finished MultiCast for PreAccept", ep.self)
			}
			ism.preAcceptNo = int(F - 1)
			ism.state = PreAccepted

		case PreAccepted: // waiting for PreAccepOKMsg
			// wait for msg non-blockingly
			if ep.verbose {
				log.Printf("Leader %d in PreAccepted Phase", ep.self)
			}
			select {
			case innerMsg := <-ism.innerChan:
				if okMsg, ok := innerMsg.(common.PreAcceptOKMsg); ok {
					// check instId is correct
					if okMsg.Id.Inst != ism.self {
						// TODO: change this fatal command
						log.Fatal("Wrong inner msg!")
					}
					// if the msg has been received from the sender, break
					if bit, ok := ism.preAcceptBook[okMsg.Sender]; bit && ok {
						break
					}
					ism.preAcceptBook[okMsg.Sender] = true
					ism.preAcceptNo--
					// compare seq and dep
					if okMsg.Inst.Seq > ism.seqOK {
						ism.chooseFast = false
						ism.seqOK = okMsg.Inst.Seq
					}
					if !compareMerge(&ism.depOK, okMsg.Inst.Deps) {
						ism.chooseFast = false
					}
					if ism.preAcceptNo == 0 {
						if ep.verbose {
							log.Printf("Leader %d received enough PreAccept Msg", ep.self)
						}
						if ism.chooseFast {
							ism.state = Committed
							if ep.verbose {
								log.Printf("Leader %d choose Fast Path", ep.self)
							}

						} else {
							ism.state = Accepted
							ism.acceptNo = int(math.Floor(float64(ep.peers) / 2))
							inst := common.Instance{
								Cmd:  cmd,
								Seq:  ism.seqOK,
								Deps: ism.depOK,
							}
							ep.array[ep.self].Pending[instId-1] = &StatefulInst{inst: inst, state: Accepted}
							// send accepted msg to replicas
							aMsg.Id = common.InstRef{Replica: ep.self, Inst: instId}
							aMsg.Inst = inst
							ism.FQuorum = ep.makeMulticast(aMsg, int64(math.Floor(float64(ep.peers)/2))) // TODO: change the number of Multicast for faster response
							if ep.verbose {
								log.Printf("Leader %d finished MultiCase for Accept", ep.self)
							}
						}
					}
				}
			case <-time.After(time.Millisecond * time.Duration(100)):
				// If not received enough OK msg, re-multicase
				if ism.preAcceptNo > 0 {
					//F := math.Floor(float64(ep.peers) / 2)
					F := ep.peers
					ism.FQuorum = ep.makeMulticast(paMsg, int64(F-1))
					ism.preAcceptNo = int(F - 1)
					ism.preAcceptBook = make(map[common.ReplicaID]bool)
				}
				if ep.verbose {
					log.Printf("Time out! Leader %d re-send MultiCast for PreAccept Msg!", ep.self)
				}
			}

		case Accepted:
			if ep.verbose {
				log.Printf("Leader %d in Accepted Phase!", ep.self)
			}
			select {
			case innerMsg := <-ism.innerChan:
				ism.acceptNo--
				if _, ok := innerMsg.(common.AcceptOKMsg); ok != true {
					break
				}
				if ism.acceptNo == 0 {
					ism.state = Committed
					if ep.verbose {
						log.Printf("Leader %d received enough Accepted Msg!", ep.self)
					}
				}
			case <-time.After(time.Millisecond * time.Duration(100)):
				// If not received enough OK msg, re-multicase
				if ism.acceptNo > 0 {
					F := math.Floor(float64(ep.peers) / 2)
					ism.FQuorum = ep.makeMulticast(aMsg, int64(F-1))
					ism.acceptNo = int(math.Floor(float64(ep.peers) / 2))
				}
				if ep.verbose {
					log.Printf("Time out! Leader %d re-send MultiCast for Accept Msg!", ep.self)
				}
			}

		case Committed:
			if ep.verbose {
				log.Printf("Leader %d in Committed Phase!", ep.self)
			}
			inst := common.Instance{
				Cmd:  cmd,
				Seq:  ism.seqOK,
				Deps: ism.depOK,
			}
			ep.array[ep.self].Pending[instId-1] = &StatefulInst{inst: inst, state: Committed}
			// send commit msg to replicas
			sendMsg := common.CommitMsg{
				Id:   common.InstRef{Replica: ep.self, Inst: instId},
				Inst: inst,
			}
			ep.makeMulticast(sendMsg, int64(ep.peers-1))
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
