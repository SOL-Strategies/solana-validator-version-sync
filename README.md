# Solana Validator Version Sync

A version synchronization manager for Solana validators that monitors the validator's current version and syncs it with the latest available versions based on SFDP requirements and GitHub releases.

## Features

- **Version Monitoring**: Continuously monitors the validator's current running version
- **SFDP Compliance**: Checks version requirements against SFDP (Solana Foundation Delegation Program) bounds
- **GitHub Integration**: Fetches available versions from GitHub releases based on cluster-specific release notes
- **Flexible Commands**: Executes configurable commands during version sync with template interpolation
- **Multiple Clients**: Supports Agave, Jito-Solana, and Firedancer validators
- **Development Tools**: Includes mock server for testing and development

## Installation

### From Source

```bash
git clone https://github.com/sol-strategies/solana-validator-version-sync.git
cd solana-validator-version-sync
make build
```

### Using Go Install

```bash
go install github.com/sol-strategies/solana-validator-version-sync/cmd/solana-validator-version-sync@latest
```

## Configuration

Create a configuration file (e.g., `config.yaml`) based on the example in `example-config.yml`:

```yaml
log:
  level: info
  format: text

validator:
  client: agave
  rpc_url: http://127.0.0.1:8899
  identities:
    active: /home/solana/active-identity.json
    passive: /home/solana/passive-identity.json

cluster:
  name: testnet

sync:
  interval_duration: 10m
  enable_sfdp_compliance: true
  client_source_repositories:
    agave:
      url: https://github.com/anza-xyz/agave
      release_notes_regexes:
        mainnet-beta: ".*This is a stable release suitable for use on Mainnet Beta.*"
        testnet: ".*This is a Testnet release.*"
  allowed_semver_changes:
    major: false
    minor: true
    patch: true
  commands:
    - name: "build validator"
      cmd: /home/solana/bin/solana-validator-source.sh
      args: ["build", "--client={{ .Sync.Client }}", "--version={{ .Sync.ToVersion }}"]
```

## Usage

### Run the Version Sync Manager

```bash
solana-validator-version-sync run --config config.yaml
```

### Command Line Options

- `--config, -c`: Path to configuration file (default: `~/solana-validator-version-sync/config.yaml`)
- `--log-level, -l`: Log level (debug, info, warn, error, fatal) - overrides config file setting

## Development

### Prerequisites

- Go 1.24 or later
- Make

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Development with Mock Server

1. Start the mock validator server:
   ```bash
   make mock-server
   ```

2. In another terminal, run the program:
   ```bash
   make dev
   ```

### Available Make Targets

- `build` - Build the binary
- `build-linux` - Build for Linux AMD64
- `clean` - Clean build artifacts
- `test` - Run tests
- `test-coverage` - Run tests with coverage
- `lint` - Run linter
- `fmt` - Format code
- `deps` - Download dependencies
- `run-demo` - Run with demo configuration
- `mock-server` - Start mock server
- `dev` - Run in development mode
- `install` - Install the binary
- `help` - Show help

## Architecture

The program is structured as follows:

- `cmd/` - Command-line interface using Cobra
- `internal/config/` - Configuration management with validation
- `internal/sync/` - Main sync logic and orchestration
- `internal/rpc/` - Solana validator RPC client
- `internal/sfdp/` - SFDP API client
- `internal/github/` - GitHub releases API client
- `internal/command/` - Command execution with template interpolation
- `demo/` - Mock server and demo configuration

## How It Works

1. **Configuration Loading**: Loads and validates the configuration file
2. **Validator State**: Queries the local validator to get current version and identity
3. **SFDP Check**: Optionally checks if the validator is in SFDP and gets version requirements
4. **Version Discovery**: Fetches available versions from GitHub releases based on cluster-specific regex patterns
5. **Sync Decision**: Determines if an upgrade, downgrade, or no change is needed
6. **Command Execution**: Executes configured commands with template interpolation for version sync

## Template Variables

Commands support template interpolation with the following variables:

- `.Hostname` - Hostname of the validator
- `.Validator.RPCURL` - RPC URL of the validator
- `.Validator.Role` - Role of the validator (active/passive)
- `.Validator.IdentityPublicKey` - Public key of the validator's identity
- `.Sync.Client` - Client name (agave, jito-solana, firedancer)
- `.Sync.FromVersion` - Current running version
- `.Sync.ToVersion` - Target version
- `.Sync.Role` - Role of the validator
- `.Sync.Cluster` - Cluster name
- `.Sync.IsSFDPComplianceEnabled` - Whether SFDP compliance is enabled

## License

This project is licensed under the MIT License - see the LICENSE file for details.
