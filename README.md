# MIT 6.824: Distributed Systems

The 6.824 is a great course on distributed systems from MIT.
Alongside the lectures, you also have a chance to implement a basic distributed system using Go by going through 4 labs.
You are going to learn and implement MapReduce, Raft, Sharding and a lot of other stuff.

Online Content:

- [Course Homepage](https://pdos.csail.mit.edu/6.824/index.html)
- [Course Video](https://www.youtube.com/playlist?list=PLrw6a1wE39_tb2fErI4-WkMbsvGQk9_UB)

How to approach it:

1. Watch the lecture videos with the handouts from the course
2. Read the papers
3. Taking notes 
4. Implement the labs(repeating the step 1 and 2)

## Notes on Lecture

There are already [official notes](https://pdos.csail.mit.edu/6.824/schedule.html) for each lecture on the course website.

I just try to note something I thought was interesting while I am watching the videos.

1. [X] [Lecture 1: Introduction](Lecture1-Introduction.md)
2. [ ] [Lecture 2: RPC and Threads]()
3. [ ] [Lecture 3: GFS]()
4. [ ] [Lecture 4: Primary-Backup Replication]()
5. [ ] [Lecture 5: Go, Threads and Raft]()
6. [X] [Lecture 6-7: Fault Tolerance Raft](Lecture6-7-Fault-Tolerance-Raft.md)
...
:construction:

## Notes on Lab

There are 4 labs in this course. 

They are supposed to be written in [Go language](https://golang.org).
Have a quick [introduction](https://tour.golang.org) here on the official website.

You can also refer to this [cheatsheet of Go](https://github.com/alfmunny/cheatsheets/blob/master/go-cheatsheet.md) for a quick recap of the basics.

1. [X] Lab 1: MapReduce [Notes](Lab1-MapReduce.md), [Code](6.824lab/src/mr) :checkered_flag:
2. [X] Lab 2: Raft [Code](6.824lab/src/raft) :checkered_flag: 
	- [X] Part 2A [Notes](Lab2-Raft-2A.md)
	- [X] Part 2B [Notes](Lab2-Raft-2B.md)
	- [X] Part 2C [Notes](Lab2-Raft-2C.md)

3. [ ] Lab 3: Fault-tolerant Key/Value Service [Notes](), [Code]()
4. [ ] Lab 4: Sharded Key/Value Service [Notes](), [Code]()
