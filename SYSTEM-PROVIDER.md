# System identity provider

This repository preserves Zitadel v4.20.0 at upstream commit
`8afa5dd37a8e723efb9eace8d09fd5c1cccef694` and its AGPL-3.0 license.
The System patch adds EdDSA to the API access-token verifier's accepted signing
algorithms. Zitadel already supports issuing these tokens. Issuer, expiry,
signature verification and API authorization remain enforced.

Corresponding source, including the patch and regression test, is available at
https://github.com/gethamster/oss-zitadel. Release artifacts identify their source
commit. Native System packages additionally apply the documented loopback-only
listener change in System's `apps/studio/deploy/identity/build-native.py`; that
source and build receipt accompany the native distribution.

Run the focused verifier check with:

```sh
pnpm install --frozen-lockfile
pnpm nx run @zitadel/api:test-unit --skipNxCache --command='go test -race ./internal/authz/repository/eventsourcing/eventstore -count=1 -v'
```

Use the upstream `@zitadel/api` Nx generation, console and packaging targets for
complete release artifacts. No database migration or principal replacement is
part of this patch.
