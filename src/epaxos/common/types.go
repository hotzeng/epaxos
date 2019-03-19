package common

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
	CmdType CmdType
	Key     Key
	Value   Value
}

type Instance struct {
	Cmd   Command
	Seq   Sequence
	NDeps int64 `struc:"sizeof=Deps"`
	Deps  []InstRef
	Mu    sync.Mutex
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
	MId int64
	Cmd Command
}
type RequestOKMsg struct {
	MId int64
	Err bool
}

type RequestAndReadMsg struct {
	MId int64
	Cmd Command
}
type RequestAndReadOKMsg struct {
	MId   int64
	Err   bool
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
	Sender ReplicaID
}

type AcceptMsg struct {
	Id   InstRef
	Inst Instance
}
type AcceptOKMsg struct {
	Id     InstRef
	Inst   Instance
	Sender ReplicaID
}

type CommitMsg struct {
	Id   InstRef
	Inst Instance
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

type KeepMsg struct {
	MId int64
}

type ProbeReqMsg struct {
	MId     int64
	Replica ReplicaID
}

type ProbeMsg struct {
	Replica      ReplicaID
	Payload      int64
	RequestReply bool
}

type ClientMsg interface {
	GetSender() int64
}

func (m RequestMsg) GetSender() int64 {
	return m.MId
}
func (m RequestOKMsg) GetSender() int64 {
	return m.MId
}
func (m RequestAndReadMsg) GetSender() int64 {
	return m.MId
}
func (m RequestAndReadOKMsg) GetSender() int64 {
	return m.MId
}
func (m KeepMsg) GetSender() int64 {
	return m.MId
}
func (m ProbeReqMsg) GetSender() int64 {
	return m.MId
}

type ServerMsg interface {
	GetSender() ReplicaID
}

func (m PreAcceptMsg) GetSender() ReplicaID {
	return m.Id.Replica
}
func (m PreAcceptOKMsg) GetSender() ReplicaID {
	return m.Id.Replica
}
func (m AcceptMsg) GetSender() ReplicaID {
	return m.Id.Replica
}
func (m AcceptOKMsg) GetSender() ReplicaID {
	return m.Id.Replica
}
func (m CommitMsg) GetSender() ReplicaID {
	return m.Id.Replica
}
func (m PrepareMsg) GetSender() ReplicaID {
	return m.Id.Replica
}
func (m TryPreAcceptMsg) GetSender() ReplicaID {
	return m.Id.Replica
}
func (m TryPreAcceptOKMsg) GetSender() ReplicaID {
	return m.Id.Replica
}
func (m ProbeMsg) GetSender() ReplicaID {
	return m.Replica
}