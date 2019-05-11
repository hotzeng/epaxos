package main

import (
	"epaxos/common"
	"log"
	"sync"
	"time"
)


type ExplicitPrepareDecision int

const (
    commit  ExplicitPrepareDecision = 0
    accept  ExplicitPrepareDecision = 1
    preAc   ExplicitPrepareDecision = 2
    noOp    ExplicitPrepareDecision = 3
    stop    ExplicitPrepareDecision = 4
    reSend  ExplicitPrepareDecision = 5
)


type rank int

const (
    noRank          rank = -1
    preAcceptRank   rank = 0
    acceptRank      rank = 1
    commitRank      rank = 2
    leaderRank      rank = 3
)


type PrepareStateMachine struct {
    ppChan  chan bool   // receive true only when get enough replies
    ppLeft  int64   // how far away from floor(N/2)
    replies []common.PrepareOKMsg
}


func NewPrepareStateMachine() *PrepareStateMachine {
	psm := new(InstStateMachine)
	return psm
}


func (ep *EPaxos) startExplicitPrepare(inst InstRef, req common.PrepareMsg) {
    targetInst := ep.array[inst.Replica].Pending[inst.Inst].inst
    targetInst.Ballot.Id = req.Ballot.Id
    ep.makeMulticast(req, ep.peers - 1)
    replyNo = 1 // skip the step of sending to ep.self, so start from replyNo=1
    psm := NewPrepareStateMachine()
    ep.psmBook[inst] = psm
    psm.ppLeft = ep.peers / 2
    var decision ExplicitPrepareDecision
    var inst common.Instance
    // replica of ballot needs to be passed because we actually not sure previously who should take care of this instance 
    for {
        decision, inst = psm.runPSM(inst, targetInst.Ballot.Replica, ep.peers / 2)
        if decision != reSend {
            break
        } else {
            ep.makeMulticast(req, ep.peers - 1)
        }
    }
    switch decision {
    case commit:
        sendReq := &common.commitMsg{}
        sendReq.Id = inst
        sendReq.Inst = targetInst
        sendReq.Inst.ballot = BallotNumber{Epoch:0, Id:req.Ballot.Id, Replica:ep.self}
        sendReq.Status = Committing
        // TODO: remain to be finished. One important change to do is that an ISM should be started, with defined state!!!

        ep.makeMulticast(sendReq, )


        }

    }
}


// in this strategy, it is slightly different from paper when there is no prepare reply with commit or accept state. In such case, if there is one preaccept, then the new leader just broadcast preaccept. If there is no preaccept, then broadcast no-operation. 
func (psm *PrepareStateMachine) runPSM(instRef InstRef, leader ReplicaID, majority int64) (decision ExplicitPrepareDecision, instToSend common.Instance) {
    instToSend = stop
    var localRank rank
    localRank = noRank
    select {
        case <-psm.ppChan: // have got enough replies
            R := common.BallotID(0)
            // find the max BallotID
            for _, okMsg := range psm.replies {
                if okMsg.Ballot.Id > R {
                    R = okMsg.Ballot.Id
                }
            }
            // begin checking all the replies, find the highest cmd
            for _, okMsg := range psm.replies {
                if ok.Msg.Ballot.Id < R {
                    continue
                }
                // if sender is leader, just stop
                if okMsg.Sender == leader {
                    // TODO: add anything?
                    return
                }
                if okMsg.State == Committing && localRank < commitRank {
                    localRank = commitRank
                }
                if okMsg.State == Accepting && localRank < acceptRank{
                    localRank = acceptRank
                }
                if okMsg.State == PreAccepting && localRank < preAcceptRank{
                    localRank = preAcceptRank
                }
            }
            if localRank == commitRank {
                instToSend = okMsg.Inst
                decision = commit
                return
            } else if localRank == acceptRank {
                instToSend = okMsg.Inst
                decision = accept
                return
            } else if localRank == preAcceptRank {
                instToSend = okMsg.Inst
                decision = accept
                return
            }
            // if does not get a pre-accept
            // TODO: when could this happen?
            decision = noOp
            return

        case <-tine.After(timeout): //TODO: timeout value may be modified
            decision = reSend
            return
    }
}

// prepare sender processes the replies
func (ep *EPaxos) ProcessPrepareOK(req common.PrepareOKMsg) {
	// The ok msg might be for different instances
    if(req.Ack) {
        psm := ep.psmBook[req.Id]
        psm.replies = append(psm.replies, req)
        psm.ppLeft--
        if ps.ppLeft == 0 {
            psm.ppChan <-true
        }
    }
}
