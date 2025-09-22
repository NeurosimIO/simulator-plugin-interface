# NeuroSim Plugin Interface

This repository defines the standard interfaces and contracts for NeuroSim simulation platform plugins. It provides binary isolation while ensuring interoperability between plugins developed by different vendors.

## Overview

The NeuroSim platform supports three types of plugins:

1. **Component Plugins** - Represent real-world systems (Train Management, Core System, Locomotive OPA, etc.)
2. **Message ICD Plugins** - Provide message libraries and encoding/decoding services
3. **Transport Plugins** - Custom transport implementations (Kafka, AMQP, Redis, etc.)

## Binary Isolation Principle

To protect intellectual property and enable multi-vendor development:

- **No Source Dependencies** - Plugins only import this public interface, never each other's code
- **gRPC Communication** - All inter-plugin communication via standardized gRPC contracts
- **Manifest-Driven** - Plugin capabilities declared through manifest files
- **Runtime Discovery** - Plugins register dynamically at runtime

## Directory Structure

```
simulator-plugin-interface/
├── proto/                          # gRPC service definitions
│   ├── plugin_service.proto        # Standard plugin interface
│   └── v1/                         # Generated Go code
│       ├── plugin_service.pb.go    # Protocol buffer types
│       └── plugin_service_grpc.pb.go # gRPC service stubs
├── client/                         # Client libraries
│   └── registration.go            # Plugin registration utilities
├── schemas/                        # JSON schema validation
│   ├── component-plugin-manifest.json
│   ├── message-icd-plugin-manifest.json
│   └── transport-plugin-manifest.json
├── examples/                       # Example manifests and code
│   ├── train-management-manifest.json
│   └── go/                        # Go implementation examples
│       ├── basic-component-plugin/
│       ├── basic-message-icd-plugin/
│       └── README.md
├── docs/                          # Documentation
│   ├── README.md                  # This file
│   ├── component-plugins.md       # Component plugin guide
│   ├── message-plugins.md         # Message ICD plugin guide
│   └── transport-plugins.md       # Transport plugin guide
├── Makefile                       # Build automation
└── go.mod                         # Go module definition
```

## Plugin Types

### Component Plugins

Component plugins represent real-world systems that participate in simulations. They:

- Implement the `PluginService` gRPC interface
- Provide component schemas for the simulation editor
- Handle message processing and instance lifecycle
- Support multiple transport options (Kafka, AMQP, etc.)

**Example**: Train Management System, Core System, Locomotive OPA

### Message ICD Plugins

Message plugins provide message libraries and encoding/decoding services. They:

- Define message types and schemas
- Encode/decode messages in various formats (JSON, binary, XML)
- Validate message structures
- Provide example payloads

**Example**: Train Management Core System ICD v0.3.3

### Transport Plugins

Transport plugins provide custom transport implementations. They:

- Handle message sending and receiving
- Support vendor-specific protocols
- Provide configuration schemas
- Enable real-world system integration

**Example**: Azure Service Bus, AWS SQS, Custom TCP protocols

## Getting Started

### For Plugin Developers

1. Import this interface in your plugin:
   ```go
   import (
       pluginpb "github.com/neurosimio/simulator-plugin-interface/proto/v1"
       "github.com/neurosimio/simulator-plugin-interface/client"
   )
   ```

2. Implement the `PluginService` interface:
   ```go
   type MyPlugin struct {
       pluginpb.UnimplementedPluginServiceServer
       manifest *pluginpb.PluginManifest
   }
   ```

3. Create your plugin manifest (see schemas/ for validation)

4. Use the registration client to register with the simulation API:
   ```go
   config := &client.RegistrationConfig{
       APIHost:     "localhost",
       APIPort:     "8080",
       Manifest:    myManifest,
       GRPCAddress: grpcAddress,
   }
   regClient := client.NewRegistrationClient(config)
   err := regClient.RegisterWithRetries(5, 2*time.Second)
   ```

5. See `examples/go/` for complete working examples

### For Integration

The simulation API uses these interfaces to:

- Discover plugin capabilities
- Route messages between plugins
- Manage plugin lifecycles
- Validate configurations

## Versioning

This interface uses semantic versioning:

- **Major**: Breaking changes to gRPC contracts
- **Minor**: New optional fields or methods
- **Patch**: Documentation and example updates

Current version: **v1.0.0**

## Binary Compatibility

Plugins compiled against this interface are guaranteed to work with:

- The same major version of the simulation API
- Other plugins using the same interface version
- Future minor/patch releases (backward compatible)

## Security

- All gRPC communication uses TLS in production
- Plugin authentication via API keys
- Manifest validation prevents malicious configurations
- Sandboxed execution environments

## Support

- **Interface Issues**: Open issues in this repository
- **Integration Help**: See vendor-specific documentation
- **API Questions**: Contact the NeuroSim platform team

## License

This interface is open source and freely available for all NeuroSim ecosystem participants.