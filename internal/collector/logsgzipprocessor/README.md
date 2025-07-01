# Logs gzip processor

The Logs gzip processor gzips the input log record body, updating the log record in-place. 

For metrics and traces, this will just be a pass-through as it does not implement related interfaces.

## Configuration

No configuration needed.

## Benchmarking

We performed benchmark measuring the performance of serial and concurrent operations (more practical) of this processor, with and without the `sync.Pool`. Here are the results:

```
Concurrent Run: Without Sync Pool
goos: darwin
goarch: arm64
pkg: github.com/nginx/agent/v3/internal/collector/logsgzipprocessor
cpu: Apple M2 Pro
BenchmarkGzipProcessor_Concurrent-12              24      45279866 ns/op    817791582 B/op     24727 allocs/op
PASS
ok      github.com/nginx/agent/v3/internal/collector/logsgzipprocessor  1.939s

Concurrent Run: With Sync Pool

goos: darwin
goarch: arm64
pkg: github.com/nginx/agent/v3/internal/collector/logsgzipprocessor
cpu: Apple M2 Pro
BenchmarkGzipProcessor_Concurrent-12             147       9383213 ns/op    10948640 B/op       7820 allocs/op
PASS
ok      github.com/nginx/agent/v3/internal/collector/logsgzipprocessor  2.026s

————

Serial Run: Without Sync Pool

goos: darwin
goarch: arm64
pkg: github.com/nginx/agent/v3/internal/collector/logsgzipprocessor
cpu: Apple M2 Pro
BenchmarkGzipProcessor/SmallRecords-12               100      12048268 ns/op    81898890 B/op       2537 allocs/op
BenchmarkGzipProcessor/MediumRecords-12              100      13143269 ns/op    82027307 B/op       2541 allocs/op
BenchmarkGzipProcessor/LargeRecords-12                91      15912399 ns/op    83198992 B/op       2580 allocs/op
BenchmarkGzipProcessor/ManySmallRecords-12             2     807707542 ns/op    8143237656 B/op   243348 allocs/op


Serial Run: With Sync Pool

goos: darwin
goarch: arm64
pkg: github.com/nginx/agent/v3/internal/collector/logsgzipprocessor
cpu: Apple M2 Pro
BenchmarkGzipProcessor/SmallRecords-12               205       7304839 ns/op     1027942 B/op        783 allocs/op
BenchmarkGzipProcessor/MediumRecords-12              182       7336266 ns/op     1078050 B/op        784 allocs/op
BenchmarkGzipProcessor/LargeRecords-12               132       9646940 ns/op     2057059 B/op        815 allocs/op
BenchmarkGzipProcessor/ManySmallRecords-12             5     239726258 ns/op     6883977 B/op      73679 allocs/op
PASS
```


To run this benchmark yourself with syncpool implementation, you can run the tests in `processor_benchmark_test.go` in with the `sync.Pool` mode. 

To compare benchmark without syncpool, you can use this code block in `processor.go` and comment the existing `gzipCompress` function, and run `processor_benchmark_test.go` :

```
func (p *logsGzipProcessor) gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
```
