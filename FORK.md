# Proxynode Xray dependency fork

This repository preserves wyx2685/Xray-core extensions on top of XTLS/Xray-core. It is the Xray kernel, not V2Ray/V2Fly or sing-box.

Initial synchronization merges official `v26.10.21` (`1b1647cd9ddf0abc34b981d4611fc03bcac1ca74`) into wyx2685 `e087e74a1bf25b46f2776d1a3b53a68f6018d654`, retaining both histories. AnyTLS, TUIC inbound, Shadowsocks changes and other fork patches are preserved. TUIC outbound remains incomplete; this synchronization does not implement it.

The official Vision compatibility fix for Go 1.27.3+ is included. Builds and CI use Go 1.27.3. Keep the original module path `github.com/xtls/xray-core`; proxynode pins this repository with a Go module replacement.

Current synchronization (2026-10-29) merges official `v26.10.29` (`0afec66085a85191e9dd2a4a6646c7c9a94688d1`). Fork protocol extensions remain unchanged. The only merge conflict was the test workflow: retain the fork's pinned Go/geodata compatibility checks rather than reintroducing upstream asset-cache jobs and repository-wide generated-code normalization. Runtime changes and the upstream protobuf tooling are included.

## Future synchronization

Fetch official release tags from `https://github.com/XTLS/Xray-core`, merge a reviewed release into this fork, and resolve actual conflicts without discarding fork-only protocols. Do not automatically merge moving upstream main. Run the fork compatibility checks and the real Mihomo/proxynode suite in `ilnli/v2node` before advancing its dependency pin or publishing proxynode binaries. Pin immutable commit IDs and record the tested Mihomo version and checksum.

This fork is consumed as a library. Automatic upstream multi-platform releases, Docker publishing and daily asset jobs are manual-only here; proxynode's own CI packages and publishes distributable binaries. Original licenses and upstream attribution remain in place.
