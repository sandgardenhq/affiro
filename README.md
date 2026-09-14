# affiro

affiro produces signatures from how a user types, using timing and structure of keystrokes rather than
*what* they type.

## Components

- [`asig`](asig/README.md): the signature-creation library
- [`cmd/affiro`](cmd/affiro/README.md): CLI that monitors keystrokes and produces signatures using `asig`

## Related repositories

Affiro spans three repositories. This one owns the algorithm and the desktop CLI; the other two
depend on it.

- **[sandgardenhq/affiro-js](https://github.com/sandgardenhq/affiro-js)** (public) — a browser port of
  `asig`, published to npm as `@sandgarden/affiro`. It captures keystrokes on a web page with no CLI
  and no extension, reimplementing the signature encoding in TypeScript. `asig` is the reference
  implementation: a change to the wire format here has to land there too, or the two stop producing
  the same signature for the same input.
- **`sandgardenhq/affiro-playground`** (private, so no link) — the verification service behind
  `https://app.affiro.com`. It imports this module to verify signatures, serves the download page for
  the binaries released here, and answers the endpoints below.

### What this repo owes the playground

Two contracts run from here into the playground, and a compiler enforces neither.

**The CLI calls the playground.** `cmd/affiro/internal/cliupdate` requests
`GET /api/v1/cli/version-check` to learn whether a build is stale, and `GET /api/v1/cli/download` to
fetch its replacement. The host defaults to `https://app.affiro.com` and is overridden with
`AFFIRO_API_BASE_URL`, which is how you point a local CLI at a local playground.

**The playground parses GoReleaser's output.** `/api/v1/cli/download` proxies GitHub releases from
this repo: it picks the asset whose name ends in `_<os>_<arch>.tar.gz` (`.zip` on Windows) and pulls
out a binary named `affiro` (`affiro.exe` on Windows). Both come straight from `archives.name_template`,
`archives.formats` and `builds.binary` in [`.goreleaser.yaml`](.goreleaser.yaml). Renaming an archive
or changing its format breaks downloads for *every* published release, not just the next one, because
the playground resolves assets at request time.

`internal/releaseassets` is a hand-synced copy of the same file in affiro-playground. `CLIFileName`,
`KnownOSes`, `KnownArches` and `BuildName` must agree across both copies. They have already drifted
in style (the playground's uses `slices.Contains` where this one loops), so diff them rather than
assuming they match.

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
