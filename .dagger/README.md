# Managed API releases

The managed Dagger controller runs the existing `@zitadel/api:test-unit` and
`@zitadel/api:build` targets with the pinned Go and Node images. Nx retains the
existing protobuf, asset, and console generation dependencies. The check keeps
Go's race-enabled unit tests. Integration and runtime qualification remain
separate requirements before promoting a System release.

The native macOS ARM64 worker uses `.dagger/native.json`. It requires Node 22,
Corepack, a Go installation capable of selecting `go1.26.10`, and Python 3.11 or
newer. The package manager version comes from `package.json`. Source credentials
remain with the controller; the recipe has no publication credentials.

Packaging reads the completed executable, verifies its CLI, and writes a stable
archive containing `bin/zitadel` and `LICENSE`. R2 publication and System assembly
reuse that retained archive. Packaging does not invoke the compiler.
