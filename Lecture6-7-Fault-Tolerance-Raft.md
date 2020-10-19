# Lecture 6-7: Fault Tolerance

Why not longest log as leader?

|    | Term |   |   |
|--- |---- |--- |--- |
| S1 | 5    | 6 | 7 |
| S2 | 5    | 8 |   |
| S3 | 5    | 8 |   |

Event:

S1 receives 6, crash, reconnect elected as leader, receives 7, crash again.

S2 elected as leader, receives log of term 8 and replicate it to S3.

Reason:

8 is already been committed, we should not use the longest log of S1.

## Election Restriction:

From 5.4.1 in paper

    Raft determines which of two logs is more up-to-date by comparing the index and term of the last entries in the logs.
    If the logs have last entries with different terms, then the log with the later term is more up-to-date.
    If the logs end with the same term, then whichever log is longer is more up-to-date.

## Fast Log backup:

AppendEntries Reply

xTerm: term of conflicting entries

xIndex: first entry of xTerm

xLen: length of log

Case 1:

    S1 4 5 5
    S2 4 6 6 6

The leader doesn't have the xTerm at all, backup to the xIndex-1

Case 2:

    S1 4 4 4
    S2 4 6 6 6

The leader has the xTerm, backup to the first conflicting index, the second index of the S1

Case 3

    4
    4 6 6 6

The leader does not have the xTerm, backs up to the first 6

## Persistence

Only three items to be presisted on disk: \`log[]\`, \`currentTerm\`, \`votedFor\`

Why not \`commitIndex\`, \`lastApplied\`?

    You can reconstruct `commitIndex` and `lastApplied` using the whole log[],
    sending them to followers and get to know which one has been committed or not.

## Snapshot

Replicate the state machine, the leader drop its log before this state.

If there is a gap between one follower and current logs of the state machine: Install Snapshot RPC

## Linearizability

> an execution history is linearizable if one can find a total order of all operations, that matches real-time (for non-overlapping ops), and in which each read sees the value from the write preceding it in the order.
