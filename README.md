# OCPP Tester

OCPP Tester is a small Go-based central system for testing OCPP charge point behavior.

## Build

Compile the central system binary:

```sh
make build
```

## Run

Start the central system:

```sh
make run
```

The server listens on:

- HTTP UI and JSON APIs: `http://localhost:8080`
- OCPP WebSocket endpoint: `ws://localhost:8081/ocpp/{chargepointid}`

## Run the Test Client

In one terminal, start the central system:

```sh
make run
```

In another terminal, start the bundled test client:

```sh
make run-client
```

The client connects as `CP_000001`, sends a boot notification, status notifications,
authorization, start transaction, meter values, stop transaction, and then heartbeats.

Open the UI to inspect the active session and OCPP log:

```text
http://localhost:8080
```