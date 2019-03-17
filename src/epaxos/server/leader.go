package main

import "os"
import "sync"
import "log"
import "net"
import "net/rpc"
import "epaxos/common"
import "main"

// EPaxos only distributes msgs from client and to Instance 
// state machines. However, the state machines will send messages
// to other replicas directly, without going through the EPaxos

// func
func ()


// func to make a new server
func Make(me int) *EPaxos {

}



func (ep *EPaxos) startServer {
    // TODO: some initializations

    ep.PreAcceptChan = make(chan PreAcceptMsg, CHAN_MAX)

    for {
        // TODO
    }

}

func (ep *EPaxos) startInstanceState(id int) {
    // ism abbreviated to instance state machine
    ism := &InstState{}
    ism.self = id
    ism.state = PreAccepted

    for {
        switch ism.state {
        case PreAccepted :
            select {
                case <- getPreAcceptOK:
                    ep.state = LeaderPreAcceptOK
                default:
            }

        case LeaderPreAcceptOK:
            select {
                case <- selectFastPath:

            }


        case LeaderAccept:


        case LeaderCommit:


        }
    }
}
