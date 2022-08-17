# Grok

[![GoDoc](https://godoc.org/github.com/trivago/grok?status.svg)](https://godoc.org/github.com/trivago/grok)
[![Build Status](https://travis-ci.org/trivago/grok.svg)](https://travis-ci.org/trivago/grok)
[![Coverage Status](https://coveralls.io/repos/github/trivago/grok/badge.svg?branch=master)](https://coveralls.io/github/trivago/grok?branch=master)
[![Go Report Card](http://goreportcard.com/badge/trivago/grok)](http:/goreportcard.com/report/trivago/grok)

This is a fork of [github.com/vjeantet/grok](https://github.com/vjeantet/grok) with improved concurrency.
This fork is not 100% API compatible but the underlying implementation is (mostly) the same.

The main intention of this fork is to get rid of all the mutexes in this library to make it scale properly when using multiple go routines. Also as grok is an extension of the regexp package the function scheme of this library should be closer to golang's regexp package.

## Changes

- All patterns have to be known at creation time
- No storage of known grok expressions (has to be done be the user, similar to the go regexp package)
- No Mutexes used anymore (this library now scales as it should)
- No Graphsort required anymore to resolve dependencies
- All known patterns text files have been converted to go maps
- Structured code to make it easier to maintain
- Added tgo.ttesting dependencies for easier to write unittests
- Fixed type hint case sensitivity and added string type
- Added []byte based functions

## Benchmarks

Original version

```text
BenchmarkNew-8                      2000        899731 ns/op      720324 B/op       3438 allocs/op
BenchmarkCaptures-8                10000        200695 ns/op        4570 B/op          5 allocs/op
BenchmarkCapturesTypedFake-8       10000        197983 ns/op        4571 B/op          5 allocs/op
BenchmarkCapturesTypedReal-8       10000        206392 ns/op        4754 B/op         16 allocs/op
BenchmarkParallelCaptures-8        10000        208389 ns/op        4570 B/op          5 allocs/op (added locally)
```

This version

```text
BenchmarkNew-8                      5000        357586 ns/op      285374 B/op       1611 allocs/op
BenchmarkCaptures-8                10000        200825 ns/op        4570 B/op          5 allocs/op
BenchmarkCapturesTypedFake-8       10000        197306 ns/op        4570 B/op          5 allocs/op
BenchmarkCapturesTypedReal-8       10000        194882 ns/op        4140 B/op         12 allocs/op
BenchmarkParallelCaptures-8        30000         55583 ns/op        4576 B/op          5 allocs/op
```

Improvements

```text
BenchmarkNew-8                     +150%
BenchmarkParallelCaptures-8        +274%
```
