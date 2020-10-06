# Lecture 1: Introduction

Distributed Systems:

-   a lot of cooperating computers, storage, MapReduce, peer-to-peer

implementaiton:

-   RPC, threads, concurrency

Performance:

-   Scalability 2x computers -> 2x throughput

Fault Tolerance:

-   Availability
-   Recoverability:
-   Replication

Consistency: Put(k, v), Get(k)->v should always get the same v from all replicates Consistency and performance are always enemies. So we have Strong or Weak versions of consistency

-   Strong: Get() must check for recent Put()
-   Weak: Get() does not yield the latest Put()

MapReduce

    Input1 -> Map -> a,1 b,1 // intermediate output
    Input2 -> Map ->     b,1
    Input3 -> Map -> a,1     c,1
                      |   |   |
                      |   |   -> Reduce -> c,1
                      |   -----> Reduce -> b,2
                      ---------> Reduce -> a,2

Import details about MapReduce

1.  master gives Map tasks to workers util all Maps complete
2.  after all Maps have finished, master hands out Reduce tasks

A intersting question in the online video lecture:

How to minimize network use? Because if the Map and Reduce get the file from other storage or another distributed storage, it still has to dealt with a lot of data traffic.

-   GFS server finds out where does the input files stores and try to run the Map directly on that machine. So you read the input from local disk. However the intermediate data goes over network once, because the Reduce workers have to collect the intermediate data from diffrent Map workers

Crash:

1.  Map crashes: Intermediate files should be re-created and call Reduce on them again.
2.  Reduce crashes: no hearbeat of worker or the task is running too long. Master call the Reduce task on other worker.
