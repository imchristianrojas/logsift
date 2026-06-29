# logsift

A fast, concurrent command-line tool for parsing and summarizing
[Common Log Format](https://en.wikipedia.org/wiki/Common_Log_Format)-style web
access logs. Point it at a log file and it reports total traffic, status-code
breakdowns, and the busiest paths — chewing through a 50 MB / 500k-line log in
about a fifth of a second.

Written in pure Go with **zero third-party dependencies** (standard library
only).

## Install

```sh
go build -o logsift .
# or run without building:
go run . --file access.log
```

## Usage

```sh
logsift --file <path> [--format table|text|json] [--workers N]
```

| Flag           | Default            | Description                                                  |
| -------------- | ------------------ | ----------------------------------------------------------- |
| `--file`       | *(required)*       | Path to the log file to analyze.                            |
| `--format`     | `table`            | Output format: `table`, `text`, or `json`.                  |
| `--workers`    | `runtime.NumCPU()` | Number of parser goroutines.                                |
| `--chunk-size` | `1MiB`             | Block size for the reader (e.g. `256KiB`, `1MiB`, `4MiB`). |

### Example

```sh
$ logsift --file access.log
METRIC       VALUE
Total lines  500000
Bad lines    1
Total bytes  1623847291

STATUS  COUNT
200     412043
301     12871
404     61092
500     13993

PATH            COUNT
/api/users      98211
/api/orders     74553
/login          51002
...
```

`--format json` emits the complete stats object (every path, not just the top
10), which is handy for piping into `jq` or another tool.

## Expected log format

Each line is matched against the combined log format:

```
127.0.0.1 - - [10/Oct/2025:13:55:36 -0700] "GET /api/users HTTP/1.1" 200 1534 "https://example.com" "Mozilla/5.0"
```

Lines that don't match are counted under **Bad lines** rather than aborting the
run, so a few malformed entries won't stop you from analyzing the rest.

## How it works

```
file ──▶ readBlocks ──▶ jobs chan ──▶ N workers ──▶ results chan ──▶ Merge ──▶ output
         (1 MiB blocks)              (parse + aggregate,
                                      one local Stats each)
```

The file is read in fixed 1 MiB blocks. Each block is split on the last newline
so workers only ever see complete lines; the trailing partial line is carried
over and glued onto the front of the next block. Every worker aggregates into
its **own** `Stats` (no shared state, no locks), and the per-worker results are
merged at the end. See [`internal/pool/largefile.go`](internal/pool/largefile.go).

### Project layout

| Path                       | Responsibility                                            |
| -------------------------- | --------------------------------------------------------- |
| `main.go`                  | Flag parsing and wiring.                                  |
| `internal/parser`          | Parse one log line into a `LogEntry`.                     |
| `internal/aggregator`      | Accumulate and merge `Stats`.                             |
| `internal/pool`            | Concurrency strategies (`Run`, `RunChunked`).             |
| `internal/output`          | Render `Stats` as table / text / JSON.                    |
| `tools/benchplot`          | Turn `go test -bench` output into a chart (see below).    |

## Benchmarks

Three strategies are benchmarked against the same 500k-line / ~50 MB log:

- **Sequential** — a single goroutine, line by line (the baseline).
- **Concurrent** — a worker pool, one line per channel send.
- **Chunked** — a worker pool fed whole 1 MiB blocks, turning millions of tiny
  channel sends into a few thousand big ones.

```
logsift: 500k-line access log

  Sequential  ████████████████████████████████████████████    1.448 s  (1.0x)
  Concurrent  ██████████████······························   457.1 ms  (3.2x)
  Chunked     ███████·····································   221.0 ms  (6.6x)
```

![benchmark chart](bench.png)

> Measured on an AMD Ryzen 7 3700X (16 threads). Your numbers will differ; run
> it yourself with the commands below.

### Why chunked wins

Both `Concurrent` and `Chunked` run the *same* per-line parser in parallel — the
difference is the unit handed across the channel. `Concurrent` sends one line
per channel op (~500k tiny synchronized handoffs) and splits every line on the
single producer goroutine. `Chunked` sends one ~1 MiB block (~10k lines) per op
(~50 handoffs) and pushes the line-splitting *into* the parallel workers. Fewer
handoffs plus more parallelized work is the whole speedup.

### Tuning the block size

`--chunk-size` controls that block. It's a trade-off, not a "bigger is better"
knob: too small and you pay per-send overhead; too large and there aren't enough
blocks to keep every worker busy, so the slowest one becomes a straggler at the
tail. The sweep below (16 threads) shows the sweet spot sits around 1–4 MiB:

```
chunk-size sweep (500k lines)

  16MiB   ████████████████████████████████████████████   507.1 ms  (1.0x)
  64KiB   ████████████████████████····················   281.2 ms  (1.8x)
  256KiB  ██████████████████████······················   256.7 ms  (2.0x)
  4MiB    ███████████████████·························   217.7 ms  (2.3x)
  1MiB    ██████████████████··························   212.7 ms  (2.4x)
```

![chunk-size sweep](bench-sweep.png)

Run the sweep yourself:

```sh
go test -bench=BenchmarkChunkedSizes -benchmem -run='^$' -count=1 .
```

### Reproducing

```sh
go test -bench=. -benchmem -run='^$' -count=1 .
```

### Rendering the chart

[`tools/benchplot`](tools/benchplot) reads `go test -bench` output on stdin,
prints the Unicode bar chart above, and can optionally write a PNG. It uses only
the standard library (`image`/`image/png` with a hand-rolled bitmap font), so it
adds no dependencies.

```sh
# ASCII chart to the terminal:
go test -bench=. -run='^$' -count=1 . | go run ./tools/benchplot

# ...and a PNG image:
go test -bench=. -run='^$' -count=1 . | \
  go run ./tools/benchplot -title "logsift: 500k-line access log" -png bench.png
```

| Flag      | Default              | Description                          |
| --------- | -------------------- | ------------------------------------ |
| `-png`    | *(off)*              | Also write a PNG bar chart here.     |
| `-title`  | `logsift benchmarks` | Chart title.                         |
| `-width`  | `44`                 | Width of the ASCII bars, in chars.   |

## Testing

```sh
go test ./...
```

The chunked reader is tested with deliberately tiny block sizes (1, 3, 7 bytes)
to force lines — and even single fields — to straddle block boundaries, which is
the easiest part of the streaming logic to get wrong.
