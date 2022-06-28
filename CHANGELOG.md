# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Types of changes:

- `Added` for new features.
- `Changed` for changes in existing functionality.
- `Removed` for now removed features.
- `Deprecated` for soon-to-be removed features.
- `Fixed` for any bug fixes.
- `Security` in case of vulnerabilities.

## [Unreleased]

### Added

- Add `SkynetAPIKey` option.

### Fixed

- Ensure custom portal URLs start with `https://`.

## [2.0.2]

### Changed

- Made transition from NebulousLabs to SkynetLabs

## [2.0.1]

### Fixed

- Fixed single-file directory uploads being uploaded as files instead of
  directories.

## [2.0.0]

### Changed

- This SDK has been updated to match Browser JS and require a client. You will
  first need to create a client and then make all API calls from this client.
- Connection options can now be passed to the client, in addition to individual
  API calls, to be applied to all API calls.
- The `defaultPortalUrl` string has been renamed to `defaultSkynetPortalUrl` and
  `defaultPortalUrl` is now a function.

### Fixed

- Fix a bug where the Content-Type was not set correctly.

## [1.1.0]

*Prior version numbers skipped to maintain parity with API.*

### Added

- Generic `Upload` and `Download` functions
- Common Options object
- API authentication
- Encryption

### Fixed

- Some upload bugs were fixed.

## [0.1.0]

### Added

- Upload and download functionality.
