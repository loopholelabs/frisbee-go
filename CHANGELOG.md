# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.7.3] - 2024-03-26

### Dependencies

- Updated `polyglot` to `v1.2.2`

### Fix

- Fixed bugs where test cases would not generate random data properly
- Fixed a bug where `ConnContext` would not get called until the first packet was read 

## [v0.7.2] - 2023-08-26

### Features

- Added `StartWithListener` function to start a server with a custom listener ([#155](https://github.com/loopholelabs/frisbee-go/issues/155))

### Fixes

- Fixed a bug where the `server` would not properly return from handling packets when the concurrency value was > 1 ([#160](https://github.com/loopholelabs/frisbee-go/issues/160))

### Changes

- Removing `trunk`

### Dependencies

- Bumping `github.com/loopholelabs/common` from `v0.4.4` to `v0.4.9`
- Bumping `github.com/loopholelabs/polyglot` from `v0.5.1` to `v0.1.2`
- Bumping `github.com/rs/zerolog` from `v1.27.0` to `v1.30.0`
- Bumping `github.com/stretchr/testify` from `v1.8.0` to `v1.8.4`
- Bumping `go.uber.org/atomic` from `v1.9.0` to `v1.11.0`
- Bumping `go.uber.org/goleak` from `v1.1.12` to `v1.2.1`

## [v0.7.1] - 2022-12-01

### Fixes

- StreamHandlers would not be registered when calling `ServeConn` because they were created during a `server.Start`
  call. This has been fixed by creating a dedicated `SetStreamHandler` method on the `Server` struct.

### Changes

- The `Server` struct now has a `SetStreamHandler` method that allows you to register a stream handler for a given
  stream name.

## [v0.7.0] - 2022-09-27

### Changes

- **[BREAKING]** The `Heartbeat` option for the Server, Client, and Async connections has been removed
- The previous heartbeat semantics have been simplified to allow for simplified bi-directional heartbeats and a lower
  overall processing overhead

### Features

- Streaming capabilities have been added to the Async connections allowing you to register a "stream" callback and the
  ability to create streams from either peer.

## [v0.6.0] - 2022-08-07 (Beta)

### Changes

- **[BREAKING]** The `server` now concurrently process incoming packets from connections by calling handler functions in
  a goroutine.
  This is done to avoid blocking the main packet processing loop when the handler for an incoming packet is slow.
- The `UPDATE` Action has been completely removed from the `server` and the `client` - the context can no longer be
  updated from a handler function.
- The `SetConcurrency` function has been added to the `server` to set the concurrency of the packet processing
  goroutines.
- `io/ioutil.Discard` has been replaced with `io.Discard` because it was being deprecated
- The `README.md` file has been updated to reflect the `frisbee-go` package name, and better direct users to the `frpc.io` website.
- @jimmyaxod has been added as a maintainer for the `frisbee-go` package.

## [v0.5.4] - 2022-07-28 (Beta)

### Features

- Renaming the `defaultDeadline` to `DefaultDeadline` and increasing the value to 5 seconds from 1 second

## [v0.5.3] - 2022-07-27 (Beta)

### Features

- Adding the `GetHandlerTable` function to the server which allows us to retrieve the handler table from the server

## [v0.5.2] - 2022-07-22 (Beta)

### Features

- Adding the `SetHandlerTable` function to the server which allows us to modify the handler table in the server

### Changes

- Close errors when the connection is already closed will now log at the Debug level

## [v0.5.1] - 2022-07-20 (Beta)

### Fixes

- Fixed an issue where new connections in the server would be overwritten sometimes due to a pointer error

### Changes

- FRPC is now called fRPC
- fRPC has been moved into its own [repository](https://github.com/loopholelabs/frpc-go)
- We're using the [Common](https://github.com/loopholelabs/common) library for our queues and packets
- Packets now use the [Polyglot-Go](https://github.com/loopholelabs/polylgot-go) library for serialization

## [v0.5.0] - 2022-05-18 (Beta)

### Changes

- Updating Frisbee RPC references to FRPC
- Updating documentation site to point to https://frpc.io
- Updating code quality according to https://deepsource.io

## [v0.4.6] - 2022-04-28 (Beta)

### Fixes

- Fixing issue where generated `decode` functions for slices would not allocate the proper memory before decoding the slice values (Issue #108)

### Changes

- Updating Trunk Linter to `v0.11.0-beta`

## [v0.4.5] - 2022-04-22 (Beta)

### Fixes

- Fixing issue where packet.Decoder would return the decoder back to the pool before decoding was complete (Issue #102)

### Changes

- Updating Trunk Linter to `v0.10.1-beta`

## [v0.4.4] - 2022-04-21 (Beta)

### Changes

- Generated RPC Code for decoding objects no longer relies on the `*packet.Packet` structure, and instead works
  with `[]byte` slices directly
- Unit test CI actions now run in parallel
- Benchmark CI actions now run in parallel
- Buf Build Dockerfile updated to use `v0.4.4`

## [v0.4.3] - 2022-04-20 (Beta)

### Fixes

- Version [v0.4.2][v0.4.2] did not embed the templates for RPC generation into frisbee, leading to runtime panics when
  generating RPC frameworks from outside the frisbee repository directory. This has now been fixed by embedding the
  templates into the compiled plugin binary file.

## [v0.4.2] - 2022-04-20 (Beta)

### Changes

- Refactored the RPC Generator to use the `text/template`
  package ([#90](https://github.com/loopholelabs/frisbee/pull/90))
- Updated `pkg/packet.Decoder` to require only a `[]byte` to create a new decoder, instead of
  a `*packet.Packet` ([#92](https://github.com/loopholelabs/frisbee/pull/92))
- Updated Trunk Linter to v0.10.0-beta ([#92](https://github.com/loopholelabs/frisbee/pull/92))

## [v0.4.1] - 2022-03-24 (Beta)

### Changes

- Using new `internal/dialer` package to handle dialing for Async and Sync connections with automatic retires
- Handling proper backoffs for accept loop in `Server` so server does not just crash when many connections are opened at
  once

### Fixes

- Fixing `SetBaseContext`, `SetOnClosed`, and `SetPreWrite` functions to not error out if a valid function is used
- Async test cases are less flaky

## [v0.4.0] - 2022-03-24 (Beta)

### Changes

- Changing `Connect` signatures and `Start` signatures for servers, and clients
- Changing the functionality of Server.`Start` so that it blocks and returns an error
- Adding `ServeConn` and `FromConn` functions for severs and clients
- Updating `protoc-gen-frisbee` to comply with the new changes
- Updating the buf.build manifest for `protoc-gen-frisbee`
- Making `baseContext`, `onClosed`, and `preWrite` hooks for the Server private, and creating `Setter` functions that
  make it impossible to set those functions to nil

### Fixes

- Fixing panics from `ConnectSync` and `ConnectAsync` functions when the connection cannot be established - it now
  returns an error properly instead

## [v0.3.2] - 2022-03-18 (Beta)

### Changes

- Swapping the lock-free Queue out with a simpler locking queue that has significantly less lock contention in scenarios
  when multiple buffers are required.
- Refactoring server and client to spawn fewer goroutines per-connection.

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

[unreleased]: https://github.com/loopholelabs/frisbee/compare/v0.7.3...HEAD
[v0.7.3]: https://github.com/loopholelabs/frisbee/compare/v0.7.1...v0.7.3
[v0.7.2]: https://github.com/loopholelabs/frisbee/compare/v0.7.1...v0.7.2
[v0.7.1]: https://github.com/loopholelabs/frisbee/compare/v0.7.0...v0.7.1
[v0.7.0]: https://github.com/loopholelabs/frisbee/compare/v0.6.0...v0.7.0
[v0.6.0]: https://github.com/loopholelabs/frisbee/compare/v0.5.4...v0.6.0
[v0.5.4]: https://github.com/loopholelabs/frisbee/compare/v0.5.3...v0.5.4
[v0.5.3]: https://github.com/loopholelabs/frisbee/compare/v0.5.2...v0.5.3
[v0.5.2]: https://github.com/loopholelabs/frisbee/compare/v0.5.1...v0.5.2
[v0.5.1]: https://github.com/loopholelabs/frisbee/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/loopholelabs/frisbee/compare/v0.4.6...v0.5.0
[v0.4.6]: https://github.com/loopholelabs/frisbee/compare/v0.4.5...v0.4.6
[v0.4.5]: https://github.com/loopholelabs/frisbee/compare/v0.4.4...v0.4.5
[v0.4.4]: https://github.com/loopholelabs/frisbee/compare/v0.4.3...v0.4.4
[v0.4.3]: https://github.com/loopholelabs/frisbee/compare/v0.4.2...v0.4.3
[v0.4.2]: https://github.com/loopholelabs/frisbee/compare/v0.4.1...v0.4.2
[v0.4.1]: https://github.com/loopholelabs/frisbee/compare/v0.4.0...v0.4.1
[v0.4.0]: https://github.com/loopholelabs/frisbee/compare/v0.3.2...v0.4.0
[v0.3.2]: https://github.com/loopholelabs/frisbee/compare/v0.3.1...v0.3.2
[v0.3.1]: https://github.com/loopholelabs/frisbee/compare/v0.3.0...v0.3.1
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
