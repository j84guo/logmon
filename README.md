# Overview
This tool reads [HTTP logs](https://en.wikipedia.org/wiki/Common_Log_Format) from stdin and prints 2 types of messages:

1. Reports
    * Printed to summarize monitored traffic every 10 seconds of runtime
    * Number of hits by website section, where section is defined as the first path component
        * Number of hits by HTTP method within each section
        * Number of hits by HTTP status within each section
2. Alerts
    * Triggered when monitored traffic exceeds some threshold over any 2 minutes of runtime
    * Resolved when monitored traffic falls under some threshold over any 2 minutes of runtime
    
# Running
I like to rate limit the transfer of log data `pv -qL 10k sample_csv.txt | go run main.go 10`.

# Testing
Tests cover the `github.com/j84guo/logmon/http/http_log_alerter.go` file.

`go test ./...`

# Design
## Language
I thought it was logical to formulate this problem in terms of concurrent, communicating tasks (user-level threads). 

I used Go because the code for spawning goroutines and communicating between them is concise, yet powerful. For example,
multiplexing multiple channel reads onto one goroutine is easy using `select`.

## Goroutines, channels
The main goroutine reads log lines from stdin and passes them to two consuming goroutines. One of these is for 
reporting, while the other is for alerting. 

Each consuming goroutine adds received logs to some internal (local to the goroutine) data structures and prints a 
message when necessary. The reporting goroutine uses a 10-second ticker to know when it should print a report. The 
alerting goroutine uses a 1-second ticker to know when it should check for and print any alerts.

Both goroutines `select` on a channel of logs and a `time.Ticker` channel. This way, they can go about updating their
internal data structures with logs in the period in between each tick, avoiding potentially delaying reports/alerts due 
to processing of a large backlog of logs that accumulated since the last tick.

The channels of logs are bounded (1024) so that slow consumption or fast production do not cause memory bloat.

## Data Structures
The reporting goroutine uses an LFU-cache like data structure called `HttpLogTracker`. It maps keys (website sections)
to integers (hit counts), allowing the hit counts to be incremented in constant time while maintaining the keys sorted
by hit count. Internally, this struct uses a doubly-linked list of buckets, each of which contains a set of keys with
the same hit count. This structure allows traffic reports to display the sections in decreasing order of traffic, 
without having to sort on each tick.

The alerting goroutine uses a queue of fixed-length 120, where each element holds the total hit count over a 1-second 
period. Thus, the sum of elements in the queue is the number of hits over 2 minutes. By sliding this 2-minute window
forwards every second, alerts can be triggered or resolved based on the provided threshold.

## Potential Improvements
* A circular buffer using contiguously-allocated memory probably has better push/pop time and space usage than a 
  doubly-linked list (more cache friendly, up-front memory allocation). Therefore, for the alerting goroutine's queue, 
  maybe Go's `container/ring.Ring` would have been more appropriate than the `container/list.List` which I used - this
  efficiency may be more important as the rate of input logs increases.
    * That said, both data structures have constant time complexity for push/pop - any improvement would be a constant
      factor/offset.
* Alerts on certain HTTP status codes seem useful, e.g. a server repeatedly crashing and returning some status 5XX is
  something developers would be interested in, since it suggests an application bug.
* The busiest server IP address is another interesting statistic. It indicates which servers might need to be 
  replicated.
    * In a similar vein, alerting based on per-server traffic would be useful.
* Better test coverage, specifically `HttpLogTracker`.
* Perhaps replacing the hand-written `HttpLogTracker` with a third-party library providing some LFU-cache-like 
  structure.
