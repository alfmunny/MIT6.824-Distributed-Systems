# Raft 2A

Readings:

- [Paper](https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf)
- [Student Guide](https://thesquareplanet.com/blog/students-guide-to-raft/)

## Raft basics

Each server has three states:

* Leader: handles all client requests
* Follower: only respond to leader and candidates
* Candidate: elect a new leader in them

**Term**:

Raft divides time into terms.

Each term begins with an election.


Term is the system's logic clock. The server can check its current term and the term from another server, to see if he is catching up.

## Leader election

Leaders send heartbeat to all other servers.

If a follower receives no heartbeat over a period of time called the election timeout, it begins an election.

Election:

1. increment its current term, convert its state to Candidate
2. vote for self
3. reset election timer
4. request votes from all other servers

A candidate wins an election if it receives votes from a majority of the servers

The election can be interrupted if the candidate receives a heartbeat from another server and the term is at least as large as the candidate's term, than the candidate recognize the leader as legitimate and returns to follower state.

If no leader in this term is elected, restart a new election after the election timeout. 

*Important*: 

We need to randomize the election timeout value for each server, so we can prevent multiple followers from converting themselves to candidates at the same time.

150-300ms was suggested in the paper, but the lab introduction says, we should set it larger, because the tests only allow 100 ms heartbeat, the election time out is too small for this heartbeat.

In the lab I have set the range in 500-600. It works fine.


I will first demonstrate the basic structure of the implementation. 

There are mainly three parts:

1. Main Loop for Leader or Follower or Candidate
2. Heartbeat
3. Election

## Code structure

Read the Figure 2. in the paper several time, make sure you understand the most of it.

Only begin to program when you have a basic understanding of the term, heartbeat process and election.

Review the Figure 2. time to time when you have problems on programming.

Define there states and some timeout constant

```go
const (
	Follower	= 0
	Candidate   = 1
	Leader		= 2
	HeartBeatInterval = 100
	ElectionTimeout = 500
	ElectionTimeoutRandomRange = 100
)
```

Fill out the Raft structure

```go
type Entry struct {
	Term int
}

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

	commitIndex int
	lastApplied int
	nextIndex	[]int
	mathIndex	[]int

	state		int
	voteCount	int
	timestamp	time.Time
	electionTimeout time.Duration
}
```
Start a goroutine for the main program, which deals with three states

```go
func (rf *Raft) Run() {
	for ; !rf.killed(); {
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
			panic("State unknown")
		}
	}
}
```

What should each role do

Every time we read or write some properties of the Raft, which can also be written and read in other routine, we should use the Mutex Lock.

``` go
func (rf *Raft) isElectionTimeout() bool {
	rf.mu.Lock()
	timestamp := rf.timestamp
	timeout := rf.electionTimeout
	rf.mu.Unlock()
	return time.Now().Sub(timestamp) > timeout
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
}

func (rf *Raft) runCandidate() {
	if rf.isElectionTimeout() {
		rf.startElection()
	}

	rf.mu.Lock()
	if rf.voteCount > len(rf.peers) / 2 {
		rf.convertTo(Leader)
	}
	rf.mu.Unlock()
}

func (rf *Raft) convertTo(state int) {
	switch state {
	case Follower:
		rf.state = Follower
		// Reset the vote if it becomes follower
		rf.votedFor = -1
	case Leader:
		rf.state = Leader
	case Candidate:
		rf.state = Candidate
	}
	DPrintf("Convert server %v from %v to %v \n", rf.me, rf.state, state)
}
```

## Implementation

### Heartbeat

```go
func (rf *Raft) sendHearbeat() {
	for i, _ := range rf.peers {
		// Do not send heartbeat to self
		if i == rf.me {
			continue
		}

		rf.mu.Lock()
		state := rf.state
		rf.mu.Unlock()

		if state != Leader {
			return
		}
		go func(index int) {
			entries := []Entry{}
			rf.mu.Lock()
			args := AppendEntriesArgs{
				Term:			rf.currentTerm,
				LeaderId:		rf.me,
				PrevLogIndex:   rf.lastApplied,
				PrevLogTerm:	rf.currentTerm,
				Entries:		entries,
				LeaderCommit:	rf.commitIndex,
			}
			rf.mu.Unlock()

			reply := AppendEntriesReply{}
			if rf.sendAppendEntries(index, &args, &reply) {
				rf.mu.Lock()
				// Figure 2.: If RPC request or response contains term T > currentTerm: set currentTerm = T, convert to follower (§5.1)
				if reply.Term > rf.currentTerm {
					rf.currentTerm = reply.Term
					rf.convertTo(Follower)
				}
				rf.mu.Unlock()
			}
		}(i)
	}
}
```

`AppendEntries`

```go
type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []Entry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Success = true

	// Figure 2.:
	// 1. Reply false if term < currentTerm (§5.1)
	// 2. If RPC request or response contains term T > currentTerm: set currentTerm = T, convert to follower (§5.1)
	
	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		return
	} 

	// Figure 2.: Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)
	if len(rf.log) >= args.PrevLogIndex {
		reply.Success = false
	} else if rf.log[args.PrevLogIndex].Term!= args.PrevLogTerm {
		reply.Success = false
	} else {
		// Figure 2. : 
		// 1. If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it (§5.3)
		// 2. Append any new entries not already in the log
		// 3. If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)

		for i, entry := range args.Entries {
			if i < len(rf.log) {
				if rf.log[i].Term != entry.Term {
					rf.log = rf.log[:i]
					rf.log = append(rf.log, entry)
				}
			} else {
				rf.log = append(rf.log, entry)
			}
		}
		if args.LeaderCommit < len(rf.log) - 1 {
			rf.commitIndex = args.LeaderCommit
		} else {
			rf.commitIndex = len(rf.log) - 1
		}
	}

	//DPrintf("Server %v handles AppendEntries, args.Term %v, rf.currentTerm %v \n", rf.me, args.Term, rf.currentTerm)

	// Figure 2.: If RPC request or response contains term T > currentTerm: set currentTerm = T, convert to follower (§5.1)
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.convertTo(Follower)
	}
	reply.Term = rf.currentTerm

	// Figure 2.: If election timeout elapses without receiving AppendEntries RPC from current leader or granting vote to candidate: convert to candidate
	rf.timestamp = time.Now()
}
```
### Election

`startElection`

```go
func (rf *Raft) startElection() {
	DPrintf("Server %v: start election\n", rf.me)
	rf.mu.Lock()
	rf.currentTerm += 1
	rf.votedFor = rf.me
	rf.timestamp = time.Now()
	rf.voteCount = 1
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
				Term:       rf.currentTerm,
				CandidateId: rf.me,
				LastLogIndex: len(rf.log) - 1,
				LastLogTerm:  rf.log[len(rf.log) - 1].Term,
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
				// Figure 2.: If votedFor is null or candidateId, and candidate’s log is at least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
					rf.currentTerm = reply.Term
					rf.convertTo(Follower)
				}
				rf.mu.Unlock()
			}
		}(i)
	}
}
```

`RequestVote`

```go
type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.VoteGranted = false
	// Figure 2.: Reply false if term < currentTerm (§5.1)
	if args.Term < rf.currentTerm {
		reply.VoteGranted = false
	}

	// Figure 2.: If RPC request or response contains term T > currentTerm: set currentTerm = T, convert to follower (§5.1)
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.convertTo(Follower)
	}

	// Figure 2.: If votedFor is null or candidateId, and candidate’s log is at least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
	if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && 
	args.LastLogIndex >= rf.lastApplied {
		rf.votedFor = args.CandidateId
		rf.timestamp = time.Now()
		reply.VoteGranted = true
		DPrintf("Server %v votes for Candidate %v", rf.me, args.CandidateId)
	} 

	reply.Term =  rf.currentTerm
}
```

That's it. Don't forget to implement the `GetState()`.


## Test

You can turn on/off the debug info in `raft/util.go`

```
Debug = 1
```

```bash
❯ go test -run 2A
2020/10/15 17:15:50 Creating raft 0
2020/10/15 17:15:50 Creating raft 1
2020/10/15 17:15:50 Creating raft 2
Test (2A): initial election ...
2020/10/15 17:15:51 Convert server 1 from 1 to 1
2020/10/15 17:15:51 Server 1: start election
2020/10/15 17:15:51 Convert server 0 from 0 to 0
2020/10/15 17:15:51 Server 0 votes for Candidate 1
2020/10/15 17:15:51 Convert server 1 from 2 to 2
2020/10/15 17:15:51 Convert server 2 from 0 to 0
2020/10/15 17:15:51 Server 2 votes for Candidate 1
  ... Passed --   3.1  3   54   13008    0
2020/10/15 17:15:53 Creating raft 0
2020/10/15 17:15:53 Creating raft 1
2020/10/15 17:15:53 Creating raft 2
Test (2A): election after network failure ...
2020/10/15 17:15:54 Convert server 0 from 1 to 1
2020/10/15 17:15:54 Server 0: start election
2020/10/15 17:15:54 Convert server 1 from 0 to 0
2020/10/15 17:15:54 Server 1 votes for Candidate 0
2020/10/15 17:15:54 Convert server 0 from 2 to 2
2020/10/15 17:15:54 Convert server 2 from 0 to 0
2020/10/15 17:15:54 Server 2 votes for Candidate 0
2020/10/15 17:15:55 Convert server 1 from 1 to 1
2020/10/15 17:15:55 Server 1: start election
2020/10/15 17:15:55 Convert server 2 from 0 to 0
2020/10/15 17:15:55 Server 2 votes for Candidate 1
2020/10/15 17:15:55 Convert server 1 from 2 to 2
2020/10/15 17:15:55 Convert server 0 from 0 to 0
2020/10/15 17:15:56 Convert server 0 from 1 to 1
2020/10/15 17:15:56 Server 0: start election
2020/10/15 17:15:56 Convert server 2 from 1 to 1
2020/10/15 17:15:56 Server 2: start election
2020/10/15 17:15:56 Server 0: start election
2020/10/15 17:15:56 Server 2: start election
2020/10/15 17:15:57 Server 0: start election
2020/10/15 17:15:57 Server 2: start election
2020/10/15 17:15:57 Server 0: start election
2020/10/15 17:15:58 Server 2: start election
2020/10/15 17:15:58 Server 0: start election
2020/10/15 17:15:58 Convert server 2 from 0 to 0
2020/10/15 17:15:58 Server 2 votes for Candidate 0
2020/10/15 17:15:58 Convert server 0 from 2 to 2
2020/10/15 17:15:58 Convert server 1 from 0 to 0
  ... Passed --   5.0  3  118   22162    0
PASS
```

To see if there are race condition to resolve.

```bash
❯ go test -race -run 2A
Test (2A): initial election ...
  ... Passed --   3.1  3   54   12900    0
Test (2A): election after network failure ...
  ... Passed --   5.5  3  138   26012    0
PASS
```

