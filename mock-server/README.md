# Mock Solana Validator Server

A mock server that simulates a Solana validator for testing the version sync functionality.

## Features

- **Configurable Identity**: Reads validator identity from keypair files
- **Configurable Health Endpoint**: Control HTTP status code and response body
- **RPC Endpoints**: Supports `getIdentity` and `getVersion` methods
- **No Role Logic**: Simplified implementation without active/passive role determination

## Configuration

The server uses a YAML configuration file with the following structure:

```yaml
server:
  port: "8899"
  
validator:
  # Path to the keypair file (relative to config file or absolute path)
  identity_keypair: "../local-test/active-identity.json"
  running_version: "1.18.0"

# Health endpoint configuration
health:
  # Response status code (200, 500, etc.)
  status_code: 200
  # Response body
  response_body: "ok"
```

## Usage

### Start the server

```bash
# Using the default config.yaml
go run main.go config.yaml

# Using a specific config file
go run main.go config-passive.yaml

# Using the Makefile (from project root)
make mock-server
```

### Test the endpoints

```bash
# Health endpoint
curl http://localhost:8899/health

# Get validator identity
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getIdentity","params":[]}' \
  http://localhost:8899/

# Get validator version
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getVersion","params":[]}' \
  http://localhost:8899/
```

## Configuration Files

- `config.yaml` - Default configuration using active keypair
- `config-passive.yaml` - Configuration using passive keypair
- `config-unhealthy.yaml` - Configuration with unhealthy health endpoint (500 status)

## Keypair Files

The server reads Solana keypair files in JSON format (array of 64 bytes representing the private key). The public key is derived from the private key and returned as the validator identity.

Example keypair files are located in `../local-test/`:
- `active-identity.json` - Active validator keypair
- `passive-identity.json` - Passive validator keypair

## Docker Support

The server includes a Dockerfile for containerized deployment:

```bash
# Build the Docker image
docker build -t mock-server .

# Run the container
docker run -p 8899:8899 mock-server
```
