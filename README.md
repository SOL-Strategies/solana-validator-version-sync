# solana-validator-version-sync
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A simple version synchronization manager for Solana validators, including [SFDP](https://solana.org/delegation-program) compliance.

Run doublezero too? Keep it up to date too with [doublezero-version-sync](https://github.com/SOL-Strategies/doublezero-version-sync)

![solanna-validator-version-sync](vhs/demo.gif)


## Features

- 👀 **Version Monitoring**: Continuously monitors the validator's current running version compared to latest available releases.
- 👮 **SFDP Compliance**: Checks version requirements against SFDP (Solana Foundation Delegation Program) bounds.
- ♻️ **Sync Commands**: Executes configurable commands when a version sync for the given validator client is required.
- ⌚ **Single-shot or recurring**: Run once or on a specified interval
- ✅ **Multiple Clients**: Supports [agave](https://github.com/anza-xyz/agave), [jito-solana](https://github.com/jito-foundation/jito-solana/), [rakurai-validator](https://github.com/rakurai-io/rakurai-validator) and [firedancer](https://github.com/firedancer-io/firedancer) validator client release monitoring.

## Installation

### From Source

```bash
git clone https://github.com/sol-strategies/solana-validator-version-sync.git
cd solana-validator-version-sync
make build
```

### Download pre-built binary

Download the latest release from the [Releases page](https://github.com/sol-strategies/solana-validator-version-sync/releases).

## Usage

### Run Once

```bash
solana-validator-version-sync --config config.yaml run
```

### Run Continuously

```bash
# can run this command as a systemd service
solana-validator-version-sync --config config.yaml run --on-interval 1h
```

## Configuration

Create a configuration file (e.g., `config.yml`) with the following options (see [config.yml](config.yml) for a working example):

```yaml
log:
  level: info  # optional, default: info, one of debug|info|warn|error|fatal
  format: text # optional, default: text, one of text|logfmt|json

validator:
  client: agave                          # required, one of agave|jito-solana|rakurai-validator|firedancer (legacy alias: rakurai)
  version_constraint: ">= 2.3.6, < 3.0.0" # required, a valid go-version semver constraint string - ref https://github.com/hashicorp/go-version
  rpc_url: http://127.0.0.1:8899         # optional, default: http:127.0.0.1:8899 - local validator rpc URL

  # Preferred: resolve the active validator identity from this vote account via getVoteAccounts.
  # The identity is considered active when the vote account is either current or delinquent.
  vote_account_pubkey: <VOTE_ACCOUNT_PUBKEY>

  # Optional passive identity. Public keys avoid reading private key material.
  identities:
    passive_pubkey: <PASSIVE_IDENTITY_PUBKEY>

cluster:
  name: testnet # required - one of mainnet-beta|testnet

sync:
  # Run sync commands even when the validator is active
  # Use with care, usually only for testnet.
  enabled_when_active: false # default: false

  # Run sync commands when no active validator is found in gossip
  # This safeguards against situations where all validators are passive and shouldn't be
  # version synced which would take them out of the would-be active validators pool
  enabled_when_no_active_leader_in_gossip: false # default: false

  # Ensure the target version satisfies SFDP requirements as reported by the API:
  # https://api.solana.org/api/epoch/required_versions
  enable_sfdp_compliance: true # default: false

  # Commands to run when there is a version change. They will run in the order they are declared.  
  # cmd, args, and environment values can be template strings and will be interpolated with the following variables:
  #  .ClusterName                 cluster the validator is running on
  #  .CommandIndex                index of the command in the commands array (zero-based)
  #  .CommandsCount               count of commands in the commands array
  #  .SyncIsSFDPComplianceEnabled true|false (value of sync.enable_sfdp_compliance)
  #  .ValidatorClient             client name (value of validator.client)
  #  .ValidatorIdentityPublicKey  public key of the validator's identity as reported by .ValidatorRPCURL
  #  .ValidatorRole               active|passive
  #  .ValidatorRoleIsActive       true|false
  #  .ValidatorRoleIsPassive      true|false
  #  .ValidatorRPCURL             RPC URL of the validator (value of validator.rpc_url)
  #  .VersionFrom                 current running version as reported by .ValidatorRPCURL
  #  .VersionTo                   sync target version (core semver only, e.g. "4.0.0")
  #  .VersionToTag                full upstream release tag for the sync target (e.g. "v4.0.0-beta.2-jito")
  commands:
    - name: "build"                                      # required - vanity name for logging purposes
      allow_failure: false                               # optional, default:false - when true, errors are logged and subsequent commands executed
      stream_output: true                                # optional, default: false - when true, command output streamed
      disabled: false                                    # optional, default: false - when true, command skipped
      inherit_environment: false                         # optional, default: false - when true, inherit parent env and overlay explicit environment values
      cmd: /home/solana/scripts/build-solana.sh          # required, supports templated string
      args: ["build", "--client={{ .ValidatorClient }}"] # optional, supports templated strings
      environment:                                       # optional, values support templated strings; set inherit_environment: true if these should augment the normal process environment
        TO_VERSION: "{{ .VersionTo }}"
    # ...
```

Exactly one active identity source must be configured:

- `validator.vote_account_pubkey` (preferred): dynamically resolves the active identity from the network.
- `validator.identities.active_pubkey`: uses a configured identity public key directly.
- `validator.identities.active`: legacy path to an active identity keypair file.

The passive identity is optional. Configure at most one of `validator.identities.passive_pubkey` or the legacy `validator.identities.passive` keypair path. When a passive identity is configured, any local identity matching neither active nor passive is treated as unknown and syncing is skipped. A configuration using only public keys does not read either keypair file.

For example, a static, file-free configuration is:

```yaml
validator:
  client: agave
  version_constraint: ">= 2.3.6, < 3.0.0"
  rpc_url: http://127.0.0.1:8899
  identities:
    active_pubkey: <ACTIVE_IDENTITY_PUBKEY>
    passive_pubkey: <PASSIVE_IDENTITY_PUBKEY>
```

Legacy `active` and `passive` keypair paths remain supported for compatibility. When either is configured, the application prints a startup warning recommending the equivalent public-key or vote-account configuration. Public IP addresses are not used for role detection because gossip addresses are not an authoritative one-to-one mapping between hosts and voting identities.

If a command defines `environment` while `inherit_environment` remains `false`, the command runs with only the explicit `environment` block and does not inherit the parent process environment. Set `inherit_environment: true` when the command depends on inherited variables such as `PATH`, `HOME`, or service-injected credentials.

## Development

### Prerequisites

- Go 1.26.5 or later
- Make
- Docker (for Docker development)

### Local Development

```bash
# Build and run locally
make build
make dev

# Build for all platforms
make build-all

# Run tests
make test

# Clean build artifacts
make clean
```

### Docker Development

```bash
# Start development environment with Docker Compose
make dev-docker

# Stop development environment
make dev-docker-stop

# Build Docker image
make docker-build
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
