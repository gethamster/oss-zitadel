# Managed API releases

This System-native recipe preserves the reviewed loopback-only API listener
patch without changing the source checkout or normal upstream releases. The
managed Dagger controller runs the existing `@zitadel/api:test-unit` and
`@zitadel/api:build` targets with the pinned Go and Node images. Nx retains the
existing protobuf, asset, and console generation dependencies. The check keeps
Go's race-enabled unit tests. Integration and runtime qualification remain
separate requirements before promoting a System release.

Linux recipes clone the controller's verified `source.bundle` and check its
commit and tree before running Nx. This retains the Git metadata required by
the existing integration package initialization, including when unit tests
import it. Source credentials remain in the controller.

The native macOS ARM64 worker uses `.dagger/native.json`. It requires Node 22,
Corepack, a Go installation capable of selecting `go1.26.10`, and Python 3.11 or
newer. Set the plain build variable `ZITADEL_NODE_HOME` to an operator-owned
Node 22 installation when the shared host uses another Node release. The
package manager version comes from `package.json`. Source credentials
remain with the controller; the recipe has no publication credentials.

Packaging reads the completed executable, verifies its CLI, and writes a stable
archive containing `bin/zitadel` and `LICENSE`. R2 publication and System assembly
reuse that retained archive. Packaging does not invoke the compiler.
