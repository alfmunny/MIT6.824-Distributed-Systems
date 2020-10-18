# Raft 2B

[Code change](https://github.com/alfmunny/MIT6.824-Distributed-Systems/commit/2d1312c4ae1e3b619a166223388e6fb65c16844e)

## Summary

1. Leader receives a command(in `Start()`). It append the command to log, and update `nextIndex` and `matchIndex`
2. Sending the new Entries to followers while sending heartbeat.
3. The follower receives the `AppendEntries`
  - If the follower has already all the logs before the new one, by checking the last log term and index with `PrevLogIndex` and `PrevLogTerm`, it only needs to append the new ones.
  - If the follower last log is inconsistent with the leader, it rejects the `AppendEntries`. The leader receives a false in reply and decrements the follower's `nextIndex`

4. When leader received success from majority of the followers, which have complete the replication of new logs, the leader allow the new logs to commit. It updates `commitIndex` and send `ApplyMsg`, also updates the `lastApplied`

## Loopholes

Most important things have been mentioned in the Student Guide and the lab notes.
However I have still gone into a frustrating one of the many loopholes.

While sending heartbeat, do not forget to check the current state of the Leader. It may have been changed when the go routine is fired.

For example, the leader 1 assigns two goroutines to send heartbeat to server 2 and 3. But they are not fired exactly at the same time. 

The goroutine for server 2 fires first, but server 2 has a higher term, it rejects the heartbeat, and turn the leader 1 to a follower.

Then the second goroutine for server 3 fires, but now actually the server 1 is already not a leader anymore. Although the go routine was assigned before, we should interrupt it immediately.

This loopholes can cause your test `TestBackup2B` to fail.

The commit of the fix:

https://github.com/alfmunny/MIT6.824-Distributed-Systems/compare/2d1312c4ae1e...256142f83146



```go
func (rf *Raft) sendHearbeat() {
	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		}

		go func(index int) {
			rf.mu.Lock()
            // check the leadership, it may have been changed when the go routine fires.

			if rf.state != Leader {
				rf.mu.Unlock()
				return
			}
			args := AppendEntriesArgs{}
            ...
            ...

			rf.mu.Unlock()

			reply := AppendEntriesReply{}
			if rf.sendAppendEntries(index, &args, &reply) {
				rf.mu.Lock()
                // check the leadership again, because sendAppendEntries take some time to return, the leadership can also be changed in between.
				if rf.state != Leader {
					rf.mu.Unlock()
					return
				}

                ...
                ...
				rf.mu.Unlock()
			}
		}(i)
	}
    ...
    ...
	rf.mu.Unlock()
}

```

## Test

```bash
❯ go test -run 2B
Test (2B): basic agreement ...
  ... Passed --   1.2  3   18    4468    3
Test (2B): RPC byte count ...
  ... Passed --   2.8  3   48  113254   11
Test (2B): agreement despite follower disconnection ...
  ... Passed --   6.3  3  126   32304    8
Test (2B): no agreement if too many followers disconnect ...
  ... Passed --   3.9  5  180   39252    4
Test (2B): concurrent Start()s ...
  ... Passed --   0.9  3   10    2618    6
Test (2B): rejoin of partitioned leader ...
  ... Passed --   3.8  3  104   25315    4
Test (2B): leader backs up quickly over incorrect follower logs ...
  ... Passed --  28.2  5 2244 2194459  103
Test (2B): RPC counts aren't too high ...
  ... Passed --   2.2  3   36    9838   12
PASS
ok      _/Users/yzhang/Projects/MIT6.824-Distributed-Systems/6.824lab/src/raft  49.405s

```

Measure the performance with `time`
```bash
❯ time go test -run 2B
Test (2B): basic agreement ...
  ... Passed --   1.1  3   16    4186    3
Test (2B): RPC byte count ...
  ... Passed --   2.8  3   50  113568   11
Test (2B): agreement despite follower disconnection ...
  ... Passed --   6.4  3  128   32839    8
Test (2B): no agreement if too many followers disconnect ...
  ... Passed --   3.9  5  180   39252    4
Test (2B): concurrent Start()s ...
  ... Passed --   1.4  3   20    5298    6
Test (2B): rejoin of partitioned leader ...
  ... Passed --   5.9  3  148   37135    4
Test (2B): leader backs up quickly over incorrect follower logs ...
  ... Passed --  24.8  5 2032 1990938  102
Test (2B): RPC counts aren't too high ...
  ... Passed --   2.7  3   48   12720   12
PASS
ok      _/Users/yzhang/Projects/MIT6.824-Distributed-Systems/6.824lab/src/raft  49.371s
go test -run 2B  2.01s user 1.49s system 6% cpu 50.026 total
```

Other interesting blogs: 

https://blog.imfing.com/2020/10/mit-6.824-lab-2-raft-part-b/
https://www.cnblogs.com/mignet/p/6824_Lab_2_Raft_2B.html
