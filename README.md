# affiro

affiro produces signatures from how a user types, using timing and structure of keystrokes rather than
*what* they type.

## Components

- [`asig`](asig/README.md): the signature-creation library
- [`cmd/affiro`](cmd/affiro/README.md): CLI that monitors keystrokes and produces signatures using `asig`

## Building

```bash
go build ./...
```

The CLI binary is built from `./cmd/affiro`:

```bash
go build -o affiro ./cmd/affiro
```

See [`cmd/affiro/README.md`](cmd/affiro/README.md) for usage and Linux Wayland setup.

## License

Apache License 2.0: see [`LICENSE`](LICENSE).
