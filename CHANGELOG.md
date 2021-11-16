# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.1.5] - 2021-11-16
### Fixes
- Frisbee `Server` and `Client` now properly use timeouts for `Async` connections everywhere
- Connection timeouts are handled using `PING` `PONG` messages at the connection level to guarantee that the connection is closed

### Changes
- The `paused` connection state has been removed as it was not being used and only causing problems
- The default read and write deadline has been reduced to 1 second, and the heartbeat time has been reduced to 500 ms

## [v0.1.4] - 2021-10-03
### Fixes
- Frisbee `Server` and `Client` now handle TLS connections properly (tested with MTLS)
- Frisbee `Server` and `Client` now return an error if the `router` provided to them is not valid
- Initial error value of the `Async.Conn` and `Sync.Conn` is `nil`

### Changes
- `ConnectSync` and `ConnectAsync` no longer have the `network` field, it is now "tcp" by default

## [v0.1.3] - 2021-07-28
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

## [v0.1.2] - 2021-06-14
### Features
- Adding `Write`, `Read`, `WriteTo`, and `ReadFrom` functionality to `frisbee.Conn` to make it compatible with `io.Copy` 
functions

### Fixes
- Improving README.md with public build status
- Improving test case stability

### Changes
- `Read` and `Write` functions are now called `ReadMessage` and `WriteMesage` respectively

## [v0.1.1] - 2021-06-04
### Fixes
- (LOOP-87,LOOP-88)-fix-message-offsets ([#23](https://github.com/loopholelabs/frisbee/issues/23))
- fixing reactor not closing and heartbeat not closing bugs

## [v0.1.0] - 2021-06-03
Initial Release of Frisbee

[Unreleased]: https://github.com/loopholelabs/frisbee/compare/v0.1.5...HEAD
[v0.1.4]: https://github.com/loopholelabs/frisbee/compare/v0.1.4...v0.1.5
[v0.1.4]: https://github.com/loopholelabs/frisbee/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/loopholelabs/frisbee/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/loopholelabs/frisbee/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/loopholelabs/frisbee/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/loopholelabs/frisbee/releases/tag/v0.1.0
