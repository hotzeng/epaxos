package common

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
	CmdType CmdType
	Key     Key
	Value   Value
}

type Instance struct {
	Cmd   Command
	Seq   Sequence
	NDeps int64 `struc:"sizeof=Deps"`
	Deps  []InstRef
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
	Id     InstRef
	Inst   Instance
	Seq    Sequence
	NDeps  int64 `struc:"sizeof=Deps"`
	Deps   []InstRef
	sender ReplicaID
}

type AcceptMsg struct {
	Id   InstRef
	Inst Instance
}
type AcceptOKMsg struct {
	Id     InstRef
	Inst   Instance
	sender ReplicaID
}

type CommitMsg struct {
	Id    InstRef
	Cmd   Command
	Seq   Sequence
	NDeps int64 `struc:"sizeof=Deps"`
	Deps  []InstRef
}

type PrepareMsg struct {
	Ballot BallotNumber
	Id     InstRef
}
type PrepareOKMsg struct {
	Ack    bool
	Ballot BallotNumber
	Id     InstRef
	Inst   Instance
}

type TryPreAcceptMsg struct {
	Id   InstRef
	Inst Instance
}
type TryPreAcceptOKMsg struct {
	Id    InstRef
	Inst  Instance
	Ack   bool
	Seq   Sequence
	NDeps int64 `struc:"sizeof=Deps"`
	Deps  []InstRef
}

type ProbeMsg struct {
	Replica      ReplicaID
	Payload      int64
	RequestReply bool
}