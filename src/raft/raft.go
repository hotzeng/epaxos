package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import "sync"
import "labrpc"
import "bytes"
import "encoding/gob"
import "fmt"
import "math/rand"
//import "log"
import "time"

const (
    debug = false
)
//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make().
//
type ApplyMsg struct {
	Index       int
	Command     interface{}
	UseSnapshot bool   // ignore for lab2; only used in lab3
	Snapshot    []byte // ignore for lab2; only used in lab3
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu              sync.Mutex
	peers           []*labrpc.ClientEnd
	persister       *Persister
	me              int // index into peers[]

	// Your data here.
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
    currentTerm     int
    votedFor        int
    logs            []LogItem

    commitIndex     int
    lastApplied     int


    status          int
    getVote         []bool
    elecWin         chan bool

    recvHeartBeats  chan bool
    killed          bool
    termChanged     bool
}

const (
    Follower = 0
    Leader  = 1
    Candidate = 2
)

type LogItem struct {
    Term    int
    Command interface{}
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here.
    term = rf.currentTerm
    isleader = (rf.status == Leader)
    if isleader{
        if debug {
            fmt.Printf("--- %d tell everyone he is leader---------------------------------------\n", rf.me)
        }
    }
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here.
	// Example:
	w := new(bytes.Buffer)
	e := gob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.logs)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	// Your code here.
	// Example:
	r := bytes.NewBuffer(data)
	d := gob.NewDecoder(r)
	d.Decode(&rf.currentTerm)
	d.Decode(&rf.votedFor)
	d.Decode(&rf.logs)
}

//
// example RequestVote RPC arguments structure.
//
type RequestVoteArgs struct {
	// Your data here.
    Term          int
    CandidateId   int
    LastLogIndex  int
    LastLogTerm   int
}

//
// example RequestVote RPC reply structure.
//
type RequestVoteReply struct {
	// Your data here.
    Term        int
    VoteGranted bool
}

//
// example RequestVote RPC handler.
//
//This function is called by the server who wants to get vote from a peer. 
//The reply is the return written by the called peer
func (rf *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here.
    rf.mu.Lock()
    defer rf.mu.Unlock()
    defer rf.persist()

    // Reply false if term < currentTerm
    if args.Term < rf.currentTerm {
        reply.VoteGranted = false
        reply.Term = rf.currentTerm
        return
    } else if args.Term > rf.currentTerm {
        rf.currentTerm = args.Term
        if debug {
            fmt.Printf("------------------------------------------------------------ %d 's term changed to %d in RequestVote\n", rf.me, rf.currentTerm)
        }
        if rf.status == Follower {
            rf.termChanged = true
        }
        // If the receiver is candidate, return to follower status
        rf.status = Follower
        rf.clearGetVote()
        //fmt.Printf("%++++++++ d becones Follower", rf.me)
        rf.votedFor = -1
    }

    lastTerm := rf.logs[len(rf.logs)-1].Term
    if(rf.votedFor == -1 || rf.votedFor == args.CandidateId) {
        if args.LastLogTerm > lastTerm ||
            (args.LastLogTerm == lastTerm && args.LastLogIndex >= (len(rf.logs) - 1)) {
            reply.VoteGranted = true
            rf.votedFor = args.CandidateId
            //rf.status = Follower
            return
        } else {
            if debug {
            fmt.Printf("args.Term: %d, rf.Term: %d, args.Idx: %d, rf.Idx: %d", args.LastLogTerm, lastTerm, args.LastLogIndex, (len(rf.logs) - 1))
            }
        }
    } else {
        if debug {
            fmt.Printf("rf.vote: %d, args.Id: %d", rf.votedFor, args.CandidateId)
        }
    }
    reply.VoteGranted = false;
    //rf.status = Follower
    return;
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// returns true if labrpc says the RPC was delivered.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not

// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args RequestVoteArgs, reply *RequestVoteReply) bool {
    if debug {
        fmt.Printf("%d send request to %d, term: %d / %d\n", rf.me, server, args.Term, rf.currentTerm)
    }
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)

    if !ok {
        //fmt.Printf("++++++++ %d request to %d failed ++++++\n", rf.me, server)
        return ok;
    }

    rf.mu.Lock();
    defer rf.mu.Unlock();

    if reply.Term > rf.currentTerm {
        rf.currentTerm = reply.Term
        if debug {
            fmt.Printf("------------------------------------------------------------ %d 's term changed to %d in sendRequestVote\n", rf.me, rf.currentTerm)
        }
        rf.status = Follower
        rf.termChanged = false
        rf.clearGetVote()
        rf.votedFor = -1
        rf.persist()
    }
    if reply.VoteGranted {
        rf.getVote[server] = true
        if debug {
            fmt.Printf("***** %d get vote from %d, got %d votes !!!\n", rf.me, server, rf.getVoteCnt())
        }
        if rf.getVoteCnt() > len(rf.peers) / 2 && rf.status != Leader {
            if debug {
                fmt.Printf("##### %d win the election #####\n", rf.me)
            }
            //fmt.Printf("##### And its term is %d!!!!!!!!!!!\n", rf.currentTerm)
            rf.status = Leader
            rf.elecWin <-true
        }
    }

	return ok
}

func (rf *Raft) getVoteCnt() int {
    cnt := 0
    for _, i := range rf.getVote {
        if i == true {
            cnt = cnt+1
        }
    }
    return cnt
}

// defind AppendEntries RPC
type AppendEntriesArgs struct {
    Term            int
    LeaderId        int
    PrevLogIndex    int
    PrevLogTerm     int
    Entries         []LogItem
    LeaderCommit    int
}

type AppendEntriesReply struct {
    Term        int
    Success     bool
}

func (rf *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.persist()
	defer rf.mu.Unlock()
    rf.mu.Unlock()
    rf.recvHeartBeats<-true
    rf.mu.Lock()
    //fmt.Printf("----------------------------------- %d get heartbeasts ################-----------------##############\n", rf.me)
    //fmt.Printf("==== Leader ID is %d ====\n", args.LeaderId)
    // reject the request if it is out of date
    if args.Term < rf.currentTerm {
        reply.Success   = false
        reply.Term      = rf.currentTerm
        //fmt.Printf("!!!!!!!! Error1 !!!!!!!!\n")
        if debug {
            fmt.Printf("args.Term is:%d\n", args.Term)
            fmt.Printf("rf.currentTerm is:%d\n", rf.currentTerm)
        }
        return
    }
    if len(rf.logs) < args.PrevLogIndex+1 ||
        rf.logs[args.PrevLogIndex].Term != args.PrevLogTerm {
        reply.Success   = false
        if debug {
            fmt.Printf("!!!!!!!! Error2 !!!!!!\n")
            fmt.Printf("!!!!!!!! rf.log.Term   is %d", rf.logs[args.PrevLogIndex].Term)
            fmt.Printf("!!!!!!!! args.PrevTerm is %d", args.PrevLogTerm)
        }
        return
    }
    // If existing entries conflicts with new ones, delete it and
    //all that follows it
    for i := 0; i < len(args.Entries); i++ {
        if(len(rf.logs) == args.PrevLogIndex+1 || rf.logs[args.PrevLogIndex+1+i].Term == args.Term ) {
            rf.logs = append(rf.logs, args.Entries[i])
        } else {
            rf.logs = rf.logs[:args.PrevLogIndex+1+i-1]
        }
    }
    // update the commitIndex
    if args.LeaderCommit > rf.commitIndex {
        if(args.LeaderCommit > len(rf.logs)-1) {
            rf.commitIndex = len(rf.logs) - 1
        } else {
            rf.commitIndex = args.LeaderCommit
        }
    }
    if rf.status == Candidate {
        rf.status = Follower
        rf.clearGetVote()
    }
    // TODO: reset the timeout
    //fmt.Printf("===== %d change heartbeats to true =====\n", rf.me)
    reply.Success = true
    //rf.startTime = time.Now()
    //rf.waitTime  = time.Millisecond * time.Duration(150 + rand.Int63n(150))
}


//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true


	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
    rf.killed = true
    rf.mu.Lock()
    rf.currentTerm  = 0
    rf.votedFor     = -1
    rf.logs         = make([]LogItem, 1) // the first index is 1, so initialize logs with one extra element
    rf.logs[0].Term = 0

    rf.commitIndex  = 0
    rf.lastApplied  = 0

    rf.status       = Follower
    //rf.getVote      = make([]bool, len(rf.peers))
    rf.clearGetVote()
    //rf.elecWin      = make(chan bool, 10)

    //rf.recvHeartBeats   = make(chan bool, 10)
    rf.termChanged      = false
    rf.mu.Unlock()
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {

	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here.
    rf.currentTerm  = 0
    rf.votedFor     = -1
    rf.logs         = make([]LogItem, 1) // the first index is 1, so initialize logs with one extra element
    rf.logs[0].Term = 0

    rf.commitIndex  = 0
    rf.lastApplied  = 0

    rf.status       = Follower
    rf.getVote      = make([]bool, len(rf.peers))
    //rf.elecWin      = false

    //rf.recvHeartBeats   = false
    rf.killed           = false
    rf.termChanged      = false

    rf.elecWin          = make(chan bool, 10)

    rf.recvHeartBeats   = make(chan bool, 10)
    //rf.startTime = time.Now()
    //rf.waitTime  = 0//time.Millisecond * 150 * (1 + Float32())
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
// TODO: make a background routine to initiate election
    if debug {
        fmt.Printf("Make server %d\n", me)
    }
    go rf.startServer()

	return rf
}

func (rf *Raft) startElec() {
    // the status must be candidate when calling this func
    //fmt.Printf("%d start election\n", rf.me)
    var args RequestVoteArgs
    args.Term = rf.currentTerm
    args.CandidateId = rf.me
    args.LastLogIndex = len(rf.logs) - 1
    args.LastLogTerm = rf.logs[len(rf.logs) - 1].Term
    //rf.status = Candidate

    replies := make([]RequestVoteReply, len(rf.peers))

    for i := 0 ; i < len(rf.peers); i++ {
        //if i == rf.me {
        //    rf.getVote[i] = true
        //    fmt.Printf("***** %d get vote from %d, got %d votes !!!\n", rf.me, i, rf.getVoteCnt())
        //    if rf.getVoteCnt() > len(rf.peers) / 2 {
        //        fmt.Printf("##### %d win the election!!!!!!!!!!!\n", rf.me)
        //        //fmt.Printf("##### And its term is %d!!!!!!!!!!!\n", rf.currentTerm)
        //        rf.status = Leader
        //    }
        //    rf.votedFor = rf.me
        //    continue
        //}
        go rf.sendRequestVote(i, args, &replies[i])
    }
}

func (rf *Raft) startServer() {
    //rand.Seed(0)
    for {
        // Stop the server when it is killed
        if rf.killed == true {
            return
        }
        switch rf.status {
        case Follower:
            if debug {
                fmt.Printf("----------------------------------- %d becomes Follower\n", rf.me)
            }
            select {
            case <-rf.recvHeartBeats:
            case <-time.After(time.Millisecond * time.Duration(rand.Intn(300) + 200)):
                rf.status = Candidate
                if !rf.termChanged {
                    rf.votedFor = -1
                }
            }

        case Leader:
            // send heartbeats
            if debug {
                fmt.Println("##### Begin to send heartbeasts! #####")
            }
            rf.sendAllHeartBeats()

        case Candidate:
            if !rf.termChanged {
                rf.currentTerm++
                rf.votedFor = -1
                if debug {
                    fmt.Printf("------------------------------------------------------------ %d 's term changed to %d in Candidate State\n", rf.me, rf.currentTerm)
                }
            }
            rf.startElec()
            select {
            case <-time.After(time.Millisecond * time.Duration(rand.Intn(1000) + 200)):
                rf.termChanged = false
            case <-rf.recvHeartBeats:
                rf.status = Follower
                if debug {
                    fmt.Printf("----------------------------------- %d becomes Follower from Candidate\n", rf.me)
                }
            case <-rf.elecWin:
                rf.status = Leader
                if debug {
                    fmt.Printf("----------------------------------- %d becomes Leader from Candidate\n", rf.me)
                }
            }
        }
    }
}

func (rf *Raft) sendAllHeartBeats() {

    //fmt.Printf("+++++ %d send heartbeats +++++\n", rf.me)
    //fmt.Printf("+++++ Its term is %d  +++++\n", rf.currentTerm)
    var args AppendEntriesArgs
    args.Term           = rf.currentTerm
    args.LeaderId       = rf.me
    args.PrevLogIndex   = 0
    args.PrevLogTerm    = rf.logs[0].Term
    args.Entries        = make([]LogItem, 0)
    args.LeaderCommit   = 0
    reply := make([]AppendEntriesReply, len(rf.peers))

    for i:=0; i < len(rf.peers); i++ {
        if i == rf.me {
            continue
        }
        if debug {
            fmt.Printf("+++++ %d Send heartbeats to %d +++++\n", rf.me, i)
        }
        go rf.sendHeartBeats(i, args, &reply[i])
    }
    //time.Sleep(time.Millisecond * time.Duration(rand.Intn(40) + 10))
    time.Sleep(time.Millisecond * 100)
}

func (rf *Raft) sendHeartBeats(server int, args AppendEntriesArgs, reply *AppendEntriesReply) {
    i := server
    //fmt.Println("+++++++++++++++++++++++++++++ Enter sendHeartBeats")
    ok := rf.peers[i].Call("Raft.AppendEntries", args, reply)
    if !ok {
        if debug {
            fmt.Printf("!!!!! heart beats to %d failed !!!!!\n", i)
        }
    }
    if reply.Success == false {
        if reply.Term == rf.currentTerm {
            if debug {
                fmt.Println("Error: reply term is same as leader's term")
            }
        }
        rf.currentTerm = reply.Term
        if debug {
            fmt.Printf("------------------------------------------------------------ %d 's term changed to %d in sendHeartBeats\n", rf.me, rf.currentTerm)
        }
        rf.termChanged = false
        rf.status = Follower
        rf.votedFor = -1
        rf.clearGetVote()
    }
}



func (rf *Raft) clearGetVote() {
    for i := range rf.getVote {
        rf.getVote[i] = false
    }
}
