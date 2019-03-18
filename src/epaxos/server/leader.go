package main

import "os"
import "sync"
import "log"
import "net"
import "net/rpc"
import "epaxos/common"
import "math"

// EPaxos only distributes msgs from client and to Instance 
// state machines. However, the state machines will send messages
// to other replicas directly, without going through the EPaxos

// Struct for instance state machine
type instStateMachine struct {
    self            int                 // which instID it is processing
    state           InstState
    innerChan       chan interface{}    // the chan used to communicate with replica

    // preAccept phase
    FQuorum         []ReplicaID         // the quorun to send preaccept
    preAcceptNo     int                 // number for received preAccept msg
    preAcceptBook   map[ReplicaID]bool  // record from which preAccept is received

    // preAcceptOK phase
    chooseFast      bool
    seqOK           Sequence            // the max seq
    depsOK          []InstRef           // the union of all deps

    // accept phase
    acceptNo        int                 // the number of received accept msg
}


// func to make a new server
func Make(me int, peers int, inbound interface) *EPaxos {
    ep := &EPaxos{}
    ep.self = me
    ep.lastInst = LastInstanceID{InstanceID:0}
    ep.array = make([]*InstList, peers)
    ep.data = make(map[common.Key]common.Value)
    ep.peers = peers
    ep.inst2Chan = make(map[common.InstanceID]ChannelID)
    ep.chanHead = 0
    ep.chanTail = 0

    ep.innerChan = make(chan interface{}, CHAN_MAX)
    ep.inbound = inbound

    // TODO: some more initialization
    ep.rpc =

    return ep
}

// func to deal with client request
func (ep *EPaxos) RpcRequest(req RequestMsg, res *RequestOKMsg) error {
    if ep.chanHead == ep.chanTail + 1 {
        log.Fatal("No free channel for new instance machine")
    }
    ep.lastInst.mu.Lock()
    localInst := ep.lastInst.InstanceID++
    ep.lastInst.mu.Unlock()

    localChan = ep.chanTail++
    ep.inst2Chan[localInst] = localChan

    go ep.startInstanceState(localInst, req, innerChan[localChan])

    for {
        reply, more := <-innerChan[localChan]
        if reply.(type) == RequestOKMsg && reply.ok != true{
            res->ok = true
            break
        }
        else if !more {
            // TODO: return error
            return ...
        }
    }
    // TODO: how to deal with error return msg?
    return ...
}

func (ep *EPaxos) ProcessPreAcceptOK(req PreAcceptOKMsg) error {
    instId := req.Id.Inst
    ep.innerChan[instId] <- req
}

func (ep *EPaxos) ProcessAcceptOK(req AcceptOKMsg) error {
    instId := req.Id.Inst
    ep.innerChan[instId] <- req
}

// help func to check the interference between two commands
func interfCmd(cmd1 Command, cmd2 Command) bool {
    return cmd1.Key == cmd2.Key
           && ( (cmd1.CmdType == CmdPut && cmd2.CmdType == CmdNoOp)
           || (cmd1.CmdType == CmdNoOp && cmd2.CmdType == CmdPut)
           || (cmd1.CmdType == CmdPut && cmd2.CmdType == CmdPut
           && cmd1.Value != cmd2.Value) )
}

// help func to compare and merge two dep sets
// TODO: make a more fast implementation
func compareMerge(dep1 *[]InstRef, dep2 []InstRef) bool { // return true if the same
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
func (ep *EPaxos) startInstanceState(instId common.InstanceID, cmd Command, innerChan chan interface{}) {
    // ism abbreviated to instance state machine
    ism := &instStateMachine{}
    ism.self = instId
    ism.innerChan = innerChan
    ism.state = Start // starting from idle state, send preaccept to F
    ism.preAcceptNo = 0
    ism.chooseFast = true
    var innerMsg interface{}

    for {
        switch ism.state {
        case Start:
            deps := []InstRef
            seqMax := 0
            for index1, oneReplica := range ep.array {
                for index2, oneInst := range oneReplica {
                    if interfCmd(oneInst.inst.Cmd, cmd) {
                        deps = append(deps, InstRef{Replica: index1, Inst: index2})
                        if oneInst.inst.Seq > seqMax {
                            seqMax = oneInst.inst.Seq
                        }
                    }
                }
            }
            seq := seqMax + 1
            inst := &Instance{}
            inst.Cmd = cmd
            // modify and save instance atomically
            inst.mu.Lock()
            inst.Seq = seq
            inst.Deps = deps
            // TODO: assign NDeps
            ep.array[ep.self].Pending[instId] = StatefulInst{inst: inst, state:PreAccepted}
            inst.mu.Unlock()
            ism.seqOK = seq
            ism.depOK = deps

            // send PreAccept to all other replicas in F
            F := math.floor(ep.peers / 2)
            sendMsg := &PreAcceptMsg{}
            sendMsg.Inst = inst
            sendMsg.Id = InstRef{ep.self, instId}
            ism.Fquorum = ep.makeMulticast(sendMsg, F-1)
            ism.preAcceptNo = F-1
            ism.state = PreAccepted


        case PreAccepted: // waiting for PreAccepOKMsg
            // wait for msg non-blockingly
            select {
                case innerMsg = <-ism.innerChan:
                    if innerMsg.(type) == PreAccepOKMsg {
                        // check instId is correct
                        if innerMsg.Id.Inst != ism.self {
                            // TODO: change this fatal command
                            log.Fatal("Wrong inner msg!")
                        }
                        // if the msg has been received from the sender, break
                        if ism.preAcceptBook[innerMsg.sender] == true {
                            break;
                        }
                        ism.preAcceptBook[innerMsg.sender] = true
                        ism.preAcceptNo--
                        // compare seq and dep
                        if innerMsg.Seq > ism.seqOK {
                            ism.chooseFast = false
                            ism.seqOK = innerMsg.Seq
                        }
                        if compareMerge(ism.depOK, innerMsg.Deps) == false {
                            ism.chooseFast = false
                        }
                        if ism.preAcceptNo == 0 {
                            if ism.chooseFast == true {
                                ism.state = Committed
                            }
                            else {
                                ism.state = Accepted
                                inst.acceptNo = math.floor(ep.peers / 2)
                                inst := &Instance{}
                                inst.Cmd = cmd
                                inst.Seq = seqOK
                                inst.Deps = depOK
                                ep.array[ep.self].Pending[instId] = StatefulInst{inst: inst, state:Accepted}
                                // send accepted msg to replicas
                                sendMsg := &AcceptMsg{}
                                sendMsg.Id = InstRef{Replica:ep.self, Inst:instId}
                                sendMsg.Inst = inst
                                ism.Fquorum = ep.makeMulticast(sendMsg, math.floor(ep.peers / 2)) // TODO: change the number of Multicast for faster response
                            }
                        }
                    }
                default:
            }

        case Accepted:
            select {
            case innerMsg := <-ism.innerChan:
                if innerMsg.(type) != AcceptOKMsg {
                    break
                }
                if --sim.acceptNo == 0 {
                    sim.state = Committed
                }
            }
            default:

        case Committed:
            inst := &Instance{}
            inst.Cmd = cmd
            inst.Seq = sim.seqOK
            inst.Deps = sim.depOK
            ep.array[ep.self].Pending[instId] = StatefulInst{inst: inst, state:Committed}
            // send commit msg to replicas
            sendMsg := &CommitMsg{}
            sendMsg.Id = InstRef{Replica:ep.self, Inst:instId}
            sendMsg.Inst = inst
            ep.makeMulticast(sendMsg, peers-1)
            sim.state = Idle
            close(innerChan)
            sim.innerChan<-RequestOKMsg{true}
            return
        }
    }
}
