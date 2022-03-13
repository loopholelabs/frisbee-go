# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.3.1] - 2022-03-12 (Beta)

### Fixes

- Fixes a `protoc-gen-frisbee` bug where the client message type didn't have the correct offset

## [v0.3.0] - 2022-03-12 (Beta)

### Changes

- Moving `github.com/loopholelabs/packet` into Frisbee under `github.com/loopholelabs/frisbee/pkg/packet`
- Added `pkg/metadata` and `pkg/content` packages to work with the `pkg/packet` package
- Added initial version of `protoc-gen-frisbee` CLI for generating an RPC Framework with Frisbee

## [v0.2.4] - 2022-03-11 (Beta)

### Changes

- Update packet to `v0.2.5`
- Added `UpdateContext` function for both the server and client

## [v0.2.3] - 2022-03-10 (Beta)

### Changes

- Default logger is now silent
- Logging connection states is now primarily done at the `Debug` level
- Update packet to `v0.2.4`
- Added `SetContext` and `Context` which allows saving and fetching a context from a `frisbee.Conn` object

## [v0.2.2] - 2022-03-09 (Beta)

### Fixes

- Closing a connection doesn't mean the other side loses data it hasn't reacted to yet

### Changes

- A `Drain` function was added to the `internal/queue` package to allow the `killGoroutines` function in `async.go` to
  drain the queue once it had closed it and once it had killed all existing goroutines
- The `heartbeat` function in `client.go` was modified to exit early if it detected that the underlying connection had
  closed
- The `async` connection type was modified to hold `stale` data once a connection is closed. The `killGoroutines`
  function will drain the `incoming` queue after killing all goroutines, and store those drained packets in
  the `async.stale` variable - future and existing ReadPacket calls will first check whether there is data available in
  the `stale` variable before they error out
- Refactored the `handlePacket` functions for both servers and clients to be clearer and avoid allocations
- The `close` function call in `Async` connections was modified to set a final write deadline for its final writer flush

## [v0.2.1] - 2022-03-02 (Beta)

### Fixes

- The `Server.ConnContext` function now runs after a TLS Handshake is completed - this means if you'd like to access
  the `tls.ConnectionState` of a `frisbee.Conn` in the `ConnContext` it is now feasible to do so

### Changes

- Frisbee packets now use a `*packet.Content` object to store byte slices. This struct will append data when the `Write`
  function is called, and can be reset using the `Reset` function. Putting a packet back in a pool automatically calls
  the `Reset` function on the content as well.
- Moving the `packet` and `metadata` packages into their own repo and referencing them

## [v0.2.0] - 2022-02-22 (Beta)

### Fixes

- `pkg/ringbuffer` (now called `pkg/queue`) no longer goes into an invalid state when it overflows
- Many new test cases for the `pkg/queue` package
- Multiple Goroutine leaks

### Changes

- Frisbee now reuses byte slices reducing GC pressure with a new \*packet.Packet interface
- `pkg/queue` now has both Blocking and Non-Blocking functionality with differing performance
- Benchmarks have been modified to be more thorough and compare performance with raw TCP Throughput
- Handler function signatures no longer differ between Server and Client routing tables, and no longer include connections being passed into the handler
- `ConnContext` and `PacketContext` functions have been added to allow the `context.Context` package to be used within Handler functions

## [v0.1.6] - 2021-11-27 (Alpha)

### Fixes

- Closing an async connection no longer leaves a goroutine running waiting for a waitgroup to finish
- All connection read and writes now have strict deadlines set

### Changes

- Test cases now run in parallel and don't require port `3000` to be free
- Frisbee no longer uses a uint32 for keeping track of state, uses an atomic bool instead
- Client now has a new `Flush` function that can be called to guarantee that the async connection is flushed
- Client no longer has the `ErrorChannel` function and instead has the `CloseChannel` function that will signal when a connection is closed
- Now require Golang `1.16`

## [v0.1.5] - 2021-11-16 (Alpha)

### Fixes

- Frisbee `Server` and `Client` now properly use timeouts for `Async` connections everywhere
- Connection timeouts are handled using `PING` `PONG` messages at the connection level to guarantee that the connection is closed

### Changes

- The `paused` connection state has been removed as it was not being used and only causing problems
- The default read and write deadline has been reduced to 1 second, and the heartbeat time has been reduced to 500 ms

## [v0.1.4] - 2021-10-03 (Alpha)

### Fixes

- Frisbee `Server` and `Client` now handle TLS connections properly (tested with MTLS)
- Frisbee `Server` and `Client` now return an error if the `router` provided to them is not valid
- Initial error value of the `Async.Conn` and `Sync.Conn` is `nil`

### Changes

- `ConnectSync` and `ConnectAsync` no longer have the `network` field, it is now "tcp" by default

## [v0.1.3] - 2021-07-28 (Alpha)

### Features

- Adding TLS functionality to Frisbee servers and clients
- Separating Frisbee Connections into Synchronous and Asynchronous connections
- Create multiplexed streams on top of existing frisbee connections (Async connections only)

### Fixes

- Frisbee `Server` and `Client` now wait for goroutines to close when they are closed
- Frisbee read loop and write (flush) loops implement deadlines

### Changes

- Frisbee Message `ID` field is now a `uint64` (which makes UUID generation easier)
- `TestStreamIOCopy` now uses `net.Conn` instead of `net.Pipe` for testing
- Removed Buffer messages (for raw data), replaced them with multiplexed streams

## [v0.1.2] - 2021-06-14 (Alpha)

### Features

- Adding `Write`, `Read`, `WriteTo`, and `ReadFrom` functionality to `frisbee.Conn` to make it compatible with `io.Copy`
  functions

### Fixes

- Improving README.md with public build status
- Improving test case stability

### Changes

- `Read` and `Write` functions are now called `ReadMessage` and `WriteMesage` respectively

## [v0.1.1] - 2021-06-04 (Alpha)

### Fixes

- (LOOP-87,LOOP-88)-fix-message-offsets ([#23](https://github.com/loopholelabs/frisbee/issues/23))
- fixing reactor not closing and heartbeat not closing bugs

## [v0.1.0] - 2021-06-03 (Alpha)

Initial Release of Frisbee

[unreleased]: https://github.com/loopholelabs/frisbee/compare/v0.3.0...HEAD
[v0.3.0]: https://github.com/loopholelabs/frisbee/compare/v0.2.4...v0.3.0
[v0.2.4]: https://github.com/loopholelabs/frisbee/compare/v0.2.3...v0.2.4
[v0.2.3]: https://github.com/loopholelabs/frisbee/compare/v0.2.2...v0.2.3
[v0.2.2]: https://github.com/loopholelabs/frisbee/compare/v0.2.1...v0.2.2
[v0.2.1]: https://github.com/loopholelabs/frisbee/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/loopholelabs/frisbee/compare/v0.1.6...v0.2.0
[v0.1.6]: https://github.com/loopholelabs/frisbee/compare/v0.1.5...v0.1.6
[v0.1.5]: https://github.com/loopholelabs/frisbee/compare/v0.1.4...v0.1.5
[v0.1.4]: https://github.com/loopholelabs/frisbee/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/loopholelabs/frisbee/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/loopholelabs/frisbee/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/loopholelabs/frisbee/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/loopholelabs/frisbee/releases/tag/v0.1.0
