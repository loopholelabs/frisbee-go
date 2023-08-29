# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.1.2] - 2023-08-26

### Fixes

- Fixes an issue where decoding certain i32 or i64 values would result in an incorrect value being returned.

## [v1.1.1] - 2023-06-12

### Fixes

- Fixing the `decode_none` Rust function which would previously crash if it was decoding from a buffer of size 0.

### Dependencies

- Bumping Typescript `@typescript-eslint/eslint-plugin` from `^5.59.10` to `^5.59.11`
- Bumping Typescript `@typescript-eslint/parser` from `^5.59.10` to `^5.59.11`
- Bumping Typescript `@types/node` from `^20.2.5` to `^20.3.0`
- Bumping Rust `serde_json` from `1.0.82` to `1.0.96`
- Bumping Rust `base64` from `0.21.0` to `0.21.2`

## [v1.1.0] - 2023-06-07

### Changes

- Merging Typescript, Golang, and Rust implementations into a single repository

[unreleased]: https://github.com/loopholelabs/scale/compare/v1.1.2...HEAD
[v1.1.2]: https://github.com/loopholelabs/scale/compare/v1.1.2
[v1.1.1]: https://github.com/loopholelabs/scale/compare/v1.1.1
[v1.1.0]: https://github.com/loopholelabs/scale/compare/v1.1.0
