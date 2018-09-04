# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [1.5.0] - 2017-04-28
### Added
- `IsSystemdService` to detect if running as a systemd service.

### Changed
- Ignore SIGPIPE for systemd, reverts #15 (#17).

## [1.4.2] - 2017-04-26
### Changed
- Exit abnormally upon SIGPIPE (#15).

## [1.4.1] - 2017-03-01
### Changed
- Fix `NewEnvironment` documentation.
- Ignore SIGPIPE for systemd (#13).

## [1.4.0] - 2016-09-10
### Added
- `BackgroundWithID` creates a new context inheriting the request ID.
- `Graceful` for Windows to make porting easy, though it does not restart.

### Changed
- Fix Windows support by [@mattn](https://github.com/mattn).
- Fix a subtle data race in `Graceful`.

## [1.3.0] - 2016-09-02
### Added
- `GoWithID` starts a goroutine with a new request tracking ID.

### Changed
- `Go` no longer issues new ID automatically.  Use `GoWithID` instead.

## [1.2.0] - 2016-08-31
### Added
- `Graceful` for network servers to implement graceful restart.
- `SystemdListeners` returns `[]net.Listener` for [systemd socket activation][activation].

### Changed
- Optimize `IDGenerator` performance.
- `Server.Handler` closes connection.
- Lower `Environment.Wait` log to debug level.

## [1.1.0] - 2016-08-24
### Added
- `IDGenerator` generates UUID-like ID string for request tracking.
- `Go` issues new request tracking ID and store it in the derived context.
- `HTTPClient`, a wrapper for `http.Client` that exports request tracking ID and logs results.
- `LogCmd`, a wrapper for `exec.Cmd` that records command execution results together with request tracking ID.

### Changed
- `HTTPServer` adds or imports request tracking ID for every request.
- `Server` adds request tracking ID for each new connection.
- Install signal handler only for the global environment.

### Removed
- `Context` method of `Environment` is removed.  It was a design flaw.

## [1.0.1] - 2016-08-22
### Changed
- Update docs.
- Use [cybozu-go/netutil](https://github.com/cybozu-go/netutil).
- Conform to cybozu-go/log v1.1.0 spec.

[activation]: http://0pointer.de/blog/projects/socket-activation.html
[Unreleased]: https://github.com/cybozu-go/cmd/compare/v1.5.0...HEAD
[1.5.0]: https://github.com/cybozu-go/cmd/compare/v1.4.2...v1.5.0
[1.4.2]: https://github.com/cybozu-go/cmd/compare/v1.4.1...v1.4.2
[1.4.1]: https://github.com/cybozu-go/cmd/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/cybozu-go/cmd/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/cybozu-go/cmd/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/cybozu-go/cmd/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/cybozu-go/cmd/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/cybozu-go/cmd/compare/v1.0.0...v1.0.1
