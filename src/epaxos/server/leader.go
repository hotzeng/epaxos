package main

import "os"
import "sync"
import "log"
import "net"
import "net/rpc"
import "epaxos/common"


func (ep *EPaxos) startServer() {
    for {
        switch ep.state {
        case LeaderIdle:
            select {
                case <- getReq:
                    ep.State = LeaderPreAccept
                default:
            }

        case LeaderPreAccept:
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
