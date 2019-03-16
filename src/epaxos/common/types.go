package common

import "os"
import "sync"

type CmdType int32
type Key int32
type Value int64
type Sequence int64
type ReplicaID int64
type InstanceID int64
type EpochID int64
type BallotID int64

const (
	CmdNoOp CmdType = 0
	CmdPut  CmdType = 1
)

type Command struct {
	Cmd   CmdType
	Key   Key
	Value Value
}

type Instance struct {
	Cmd  Command
	Seq  Sequence
	Deps []InstRef
}

type InstRef struct {
	Replica ReplicaID
	Inst    InstanceID
}

type BallotNumber struct {
	Epoch   EpochID
	Id      BallotID
	Replica ReplicaID
}

type RequestMsg struct {
	Cmd Command
}
type RequestOKMsg struct {
	// Empty
}

type RequestAndReadMsg struct {
	Cmd Command
}
type RequestAndReadOKMsg struct {
	Exist bool
	Value Value
}

type PreAcceptMsg struct {
	Id   InstRef
	Inst Instance
}
type PreAcceptOKMsg struct {
	seq  Sequence
	deps []InstRef
}

type AcceptMsg struct {
	Id   InstRef
	Inst Instance
}
type AcceptOKMsg struct {
	// Empty
}

type CommitMsg struct {
	Id   InstRef
	Cmd  Command
	Seq  Sequence
	Deps []InstRef
}
type CommitOKMsg struct {
	// Empty
}

type PrepareMsg struct {
	Ballot BallotNumber
	Id     InstRef
}
type PrepareOKMsg struct {
	Ack    bool
	Ballot BallotNumber
	Inst   Instance
}

type TryPreAcceptMsg struct {
	Id   InstRef
	Inst Instance
}
type TryPreAcceptOKMsg struct {
	Ack  bool
	Seq  Sequence
	Deps []InstRef
}

type InstList struct {
	Mu      sync.Mutex
	LogFile *os.File
	Offset  InstanceID
	Pending []Instance
}

type EPaxos struct {
	Self     ReplicaID
	LastInst InstanceID
	Array    map[ReplicaID]*InstList
	Data     map[Key]Value
}
