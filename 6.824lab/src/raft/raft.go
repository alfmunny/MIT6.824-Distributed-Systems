package raft

// Leader Server 2: 11
// 0, 3: 41
// 1, 4: 43

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
import "sync/atomic"
import "../labrpc"
import "time"
import "math/rand"

import "bytes"
import "../labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm int
	votedFor    int
	log         []Entry // Entries
	applyCh     chan ApplyMsg

	commitIndex int
	lastApplied int
	nextIndex   []int
	matchIndex  []int

	state           int
	voteCount       int
	timestamp       time.Time
	electionTimeout time.Duration
}

const (
	Follower                   = 0
	Candidate                  = 1
	Leader                     = 2
	HeartBeatInterval          = 100
	ElectionTimeout            = 500
	ElectionTimeoutRandomRange = 100
)

type Entry struct {
	Term    int
	Command interface{}
	Index   int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	term = rf.currentTerm
	isleader = rf.state == Leader
	rf.mu.Unlock()
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.log)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var log []Entry
	var currentTerm int
	var votedFor int
	if d.Decode(&log) != nil ||
		d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil {
		panic("Decode error")
	} else {
		rf.log = log
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
	}
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example equestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()

	reply.VoteGranted = false

	if args.Term < rf.currentTerm {
		reply.VoteGranted = false
	}

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.convertTo(Follower)
	}

	if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && moreUpToDate(args, rf) {
		rf.votedFor = args.CandidateId
		rf.timestamp = time.Now()
		reply.VoteGranted = true
		DPrintf("Server %v votes for Candidate %v, args.LastLogTerm: %v, raft last log term: %v", rf.me, args.CandidateId, args.LastLogTerm, rf.log[len(rf.log)-1].Term)
	}
	reply.Term = rf.currentTerm
}

func moreUpToDate(args *RequestVoteArgs, rf *Raft) bool {
	// 5.4.1 Election restriction
	// If the logs have last entries with different terms, then the log with the later term is more up-to-date.
	// If the logs end with the same term, then whichever log is longer is more up-to-date.
	if args.LastLogTerm > rf.log[len(rf.log)-1].Term ||
		(args.LastLogTerm == rf.log[len(rf.log)-1].Term &&
			args.LastLogIndex >= rf.log[len(rf.log)-1].Index) {
		return true
	} else {
		return false
	}
}

func (rf *Raft) convertTo(state int) {
	switch state {
	case Follower:
		DPrintf("Convert server %v from %v to %v on term %v \n", rf.me, rf.state, "Follower", rf.currentTerm)
		rf.state = Follower
		rf.votedFor = -1
	case Leader:
		DPrintf("Convert server %v from %v to %v on term %v \n", rf.me, rf.state, "Leader", rf.currentTerm)
		rf.state = Leader
	case Candidate:
		DPrintf("Convert server %v from %v to %v on term %v \n", rf.me, rf.state, "Candidate", rf.currentTerm)
		rf.state = Candidate
	}
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
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	LeaderCommit int
	Entries      []Entry
}

type AppendEntriesReply struct {
	Term    int
	Success bool
	XTerm   int
	XIndex  int
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()

	reply.Success = true

	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		return
	}

	if args.PrevLogIndex >= len(rf.log) {
		reply.Success = false
		reply.XIndex = len(rf.log)
		reply.XTerm = -1
	} else if rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.Success = false
		reply.XTerm = rf.log[args.PrevLogIndex].Term
		XIndex := args.PrevLogIndex
		for rf.log[XIndex-1].Term == reply.XTerm {
			XIndex--
		}
		reply.XIndex = XIndex
	} else {
		if len(args.Entries) > 0 {
			for _, entry := range args.Entries {
				if entry.Index < len(rf.log) {
					if rf.log[entry.Index].Term != entry.Term {
						rf.log = rf.log[:entry.Index]
						rf.log = append(rf.log, entry)
					}
				} else {
					rf.log = append(rf.log, entry)
				}
			}

			rf.nextIndex[rf.me] = len(rf.log)
			rf.matchIndex[rf.me] = len(rf.log) - 1
			DPrintf("Server %v: After replicate nextIndex: %v, matchIndex: %v, commitIndex: %v, lastApplied %v, LastLogTerm: %v \n", rf.me, rf.nextIndex[rf.me], rf.matchIndex[rf.me], rf.commitIndex, rf.lastApplied, rf.log[len(rf.log)-1].Term)

		}
		if args.LeaderCommit > rf.commitIndex {
			if args.LeaderCommit < len(rf.log)-1 {
				rf.commitIndex = args.LeaderCommit
			} else {
				rf.commitIndex = len(rf.log) - 1
			}
		}
	}

	//DPrintf("Server %v handles AppendEntries, args.Term %v, rf.currentTerm %v \n", rf.me, args.Term, rf.currentTerm)
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.convertTo(Follower)
	}

	if rf.commitIndex > rf.lastApplied {
		rf.sendApplyMsg()
	}

	reply.Term = rf.currentTerm

	rf.timestamp = time.Now()
}

func (rf *Raft) sendApplyMsg() {
	if rf.commitIndex > rf.lastApplied {
		entries := rf.log[rf.lastApplied+1 : rf.commitIndex+1]

		go func(entries []Entry) {
			for _, entry := range entries {
				msg := ApplyMsg{
					CommandValid: true,
					Command:      entry.Command,
					CommandIndex: entry.Index,
				}

				rf.applyCh <- msg

				rf.mu.Lock()
				if rf.lastApplied < msg.CommandIndex {
					rf.lastApplied = msg.CommandIndex
				}
				rf.mu.Unlock()
			}
		}(entries)
	}
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
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

	// Your code here (2B).
	term, isLeader = rf.GetState()
	DPrintf("Receiving command %v\n", command)

	if isLeader {
		rf.mu.Lock()
		index = len(rf.log)
		entry := Entry{Term: term, Command: command, Index: index}
		DPrintf("New Entry %v\n", entry.Index)
		rf.log = append(rf.log, entry)
		rf.persist()
		rf.nextIndex[rf.me] = len(rf.log)
		rf.matchIndex[rf.me] = len(rf.log) - 1
		rf.mu.Unlock()
	}

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) isElectionTimeout() bool {
	rf.mu.Lock()
	timestamp := rf.timestamp
	timeout := rf.electionTimeout
	rf.mu.Unlock()
	return time.Now().Sub(timestamp) > timeout
}

func (rf *Raft) Run() {
	for !rf.killed() {
		rf.mu.Lock()
		state := rf.state
		rf.mu.Unlock()
		switch state {
		case Leader:
			rf.runLeader()
		case Follower:
			rf.runFollower()
		case Candidate:
			rf.runCandidate()
		default:
			panic("State unknow")
		}
	}
}

func (rf *Raft) runLeader() {
	rf.sendHearbeat()
	time.Sleep(time.Duration(HeartBeatInterval) * time.Millisecond)
}

func (rf *Raft) runFollower() {
	if rf.isElectionTimeout() {
		rf.mu.Lock()
		rf.convertTo(Candidate)
		rf.mu.Unlock()
		rf.startElection()
	}
	time.Sleep(10 * time.Millisecond)
}

func (rf *Raft) runCandidate() {
	if rf.isElectionTimeout() {
		rf.startElection()
	}

	rf.mu.Lock()
	if rf.voteCount > len(rf.peers)/2 {
		rf.convertTo(Leader)
	}
	rf.mu.Unlock()

	time.Sleep(10 * time.Millisecond)
}

func (rf *Raft) sendHearbeat() {
	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		}

		go func(index int) {
			rf.mu.Lock()
			if rf.state != Leader {
				rf.mu.Unlock()
				return
			}
			args := AppendEntriesArgs{}
			args.Term = rf.currentTerm
			args.LeaderId = rf.me
			args.LeaderCommit = rf.commitIndex
			prevLogIndex := rf.nextIndex[index] - 1
			args.PrevLogIndex = prevLogIndex
			args.PrevLogTerm = rf.log[prevLogIndex].Term
			entries := make([]Entry, len(rf.log[(prevLogIndex+1):]))
			copy(entries, rf.log[(prevLogIndex+1):])
			args.Entries = entries

			rf.mu.Unlock()

			reply := AppendEntriesReply{}
			if rf.sendAppendEntries(index, &args, &reply) {
				rf.mu.Lock()

				if rf.state != Leader {
					rf.mu.Unlock()
					return
				}

				if reply.Term > rf.currentTerm {
					rf.currentTerm = reply.Term
					rf.convertTo(Follower)
					rf.persist()
				} else if !reply.Success {
					rf.nextIndex[index] = reply.XIndex
					if reply.XTerm != -1 {
						for i := args.PrevLogIndex; i >= 1; i-- {
							if rf.log[i-1].Term == reply.XTerm {
								rf.nextIndex[index] = i
								break
							}
						}
					}
				} else {
					// rf.matchIndex[index] = len(rf.log) - 1
					// rf.nextIndex[index] = len(rf.log)
					// This is wrong, since the rf.log may have been changed after sendAppendEntries responsed
					// You can't pass the Figure 8 unreliable, the nextwork loses a lot of packets. You don't always got the response immediately.
					rf.matchIndex[index] = args.PrevLogIndex + len(args.Entries)
					rf.nextIndex[index] = rf.matchIndex[index] + 1
				}
				rf.mu.Unlock()
			}
		}(i)
	}

	rf.mu.Lock()
	if rf.commitIndex < rf.matchIndex[rf.me] {
		counter := 0
		minN := rf.matchIndex[rf.me]
		DPrintf("matchIndex on Leader %v \n", rf.matchIndex[rf.me])
		for i, _ := range rf.peers {
			if rf.matchIndex[i] > rf.commitIndex {
				DPrintf("matchIndex on server %v: %v \n", i, rf.matchIndex[i])
				counter += 1
				if rf.matchIndex[i] < minN {
					minN = rf.matchIndex[i]
				}
			}
		}

		if counter > len(rf.peers)/2 && rf.log[minN].Term == rf.currentTerm {
			rf.commitIndex = minN
			rf.sendApplyMsg()
		}
		DPrintf("Setting commitIndex on Leader to %v\n", rf.commitIndex)
	}

	rf.mu.Unlock()
}

func (rf *Raft) startElection() {
	DPrintf("Server %v: start election\n", rf.me)
	rf.mu.Lock()
	rf.currentTerm += 1
	rf.votedFor = rf.me
	rf.timestamp = time.Now()
	rf.voteCount = 1
	rf.persist()
	rf.mu.Unlock()

	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		}

		rf.mu.Lock()
		state := rf.state
		rf.mu.Unlock()

		if state != Candidate {
			return
		}

		go func(index int) {
			rf.mu.Lock()
			args := RequestVoteArgs{
				Term:         rf.currentTerm,
				CandidateId:  rf.me,
				LastLogIndex: len(rf.log) - 1,
				LastLogTerm:  rf.log[len(rf.log)-1].Term,
			}
			rf.mu.Unlock()
			reply := RequestVoteReply{}
			//DPrintf("Candidate %v Sending RequestVote to server %v\n", rf.me, index)
			if rf.sendRequestVote(index, &args, &reply) {
				//DPrintf("Candidate %v Receiving RequestVote from server %v, VoteGranted: %v\n", rf.me, index, reply.VoteGranted)
				rf.mu.Lock()
				if reply.VoteGranted {
					rf.voteCount += 1
				} else if reply.Term > rf.currentTerm {
					rf.currentTerm = reply.Term
					rf.convertTo(Follower)
					rf.persist()
				}
				rf.mu.Unlock()
			}
		}(i)
	}
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
	rf.applyCh = applyCh
	rf.nextIndex = make([]int, len(peers))
	for i, _ := range rf.peers {
		rf.nextIndex[i] = 1
	}
	rf.matchIndex = make([]int, len(peers))
	rf.commitIndex = 0
	rf.lastApplied = 0

	// Your initialization code here (2A, 2B, 2C).
	rf.timestamp = time.Now()
	rf.state = Follower
	rf.votedFor = -1
	rf.electionTimeout = time.Duration(rand.Intn(ElectionTimeoutRandomRange)+ElectionTimeout) * time.Millisecond
	// log entries; each entry contains command for state machine, and term when entry was received by leader (first index is 1)
	rf.log = append(rf.log, Entry{Term: 0})
	DPrintf("Creating raft %v", rf.me)
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	go rf.Run()

	return rf
}
