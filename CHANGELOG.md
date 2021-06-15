# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- (LOOP-87,LOOP-88)-fix-message-offsets ([#23](https://github.com/Loophole-Labs/frisbee/issues/23))
- fixing reactor not closing and heartbeat not closing bugs

## [v0.1.0] - 2021-06-03
Initial Release of Frisbee

[Unreleased]: https://github.com/olivierlacan/keep-a-changelog/compare/v0.1.2...HEAD
[v0.1.2]: https://github.com/Loophole-Labs/frisbee/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/Loophole-Labs/frisbee/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/Loophole-Labs/frisbee/releases/tag/v0.1.0
