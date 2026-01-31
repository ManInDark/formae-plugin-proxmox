# Formae Plugin Proxmox

My attempt at writing a formae plugin to interact with proxmox.

Very much WIP.

## Installation

```bash
# Install the plugin
make install
```

## Supported Resources

*TODO: Document your supported resource types*

| Resource Type                  | Description                                           |
| ------------------------------ | ----------------------------------------------------- |
| `PROXMOX::Service::LXC`        | LXC Container                                         |

## Configuration

Configure a target in your Forma file:

```pkl
new formae.Target {
    label = "proxmox"
    namespace = "PROXMOX"
    config = new Mapping {
      ["url"] = "https://your-url:8006"
      ["node"] = "nodename"
    }
  }
```

## Examples

See the [examples/](examples/) directory for usage examples.

```bash
# Evaluate an example
formae eval examples/basic/main.pkl

# Apply resources
formae apply --mode reconcile --watch examples/basic/main.pkl
```

## Development

### Prerequisites

- Go 1.25+
- [Pkl CLI](https://pkl-lang.org/main/current/pkl-cli/index.html)
- Cloud provider credentials (for conformance testing)

### Building

```bash
make build      # Build plugin binary
make test       # Run unit tests
make lint       # Run linter
make install    # Build + install locally
```

### Local Testing

```bash
# Install plugin locally
make install

# Start formae agent
formae agent start

# Apply example resources
formae apply --mode reconcile --watch examples/basic/main.pkl
```

### Conformance Testing

Conformance tests validate your plugin's CRUD lifecycle using the test fixtures in `testdata/`:

| File                     | Purpose                          |
| ------------------------ | -------------------------------- |
| `resource.pkl`           | Initial resource creation        |
| `resource-update.pkl`    | In-place update (mutable fields) |
| `resource-replace.pkl`   | Replacement (createOnly fields)  |

The test harness sets `FORMAE_TEST_RUN_ID` for unique resource naming between runs.

```bash
make conformance-test                  # Latest formae version
make conformance-test VERSION=0.80.0   # Specific version
```

The `scripts/ci/clean-environment.sh` script cleans up test resources. It runs before and after conformance tests and should be idempotent.

## Licensing

Plugins are independent works and may be licensed under any license of the author’s choosing.

See the formae plugin policy:
<https://docs.formae.io/plugin-sdk/
