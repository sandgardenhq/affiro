# client

`client` is the Go client for the affiro API. It lets a program holding an API key
upload a signed document and read the analysis back.

API keys are issued from the website's API keys page.

## Usage

```go
cl, err := client.New(os.Getenv("AFFIRO_API_KEY"))
if err != nil {
	return err
}

analysis, err := cl.Upload(ctx, signature, strings.NewReader(document))
if err != nil {
	return err
}
fmt.Println(analysis.HumanWritten) // likely, possible, unlikely, very unlikely, unknown
```

A later draft becomes a new version of the same document:

```go
_, err = cl.UploadVersion(ctx, analysis.DocumentID, signature, strings.NewReader(revision))
```

The signature comes from [`asig`](../asig/README.md).

## Failures

Every error is a `*client.Error` that unwraps to one of three sentinels, so a caller can
tell them apart with `errors.Is` rather than by reading the message:

| Sentinel | Means | Worth retrying |
|---|---|---|
| `ErrUnauthorized` | The API key was refused: unknown, revoked, or not allowed to write here. Also what `New` returns when given no key at all. | No |
| `ErrInvalidDocument` | The document could not be stored: the API refused to keep it, or this client refused to send it. | No |
| `ErrUnavailable` | The API could not be reached or could not answer. | Yes |

`*client.Error` also carries `StatusCode` and a `Message`. `StatusCode` is `0` when there was no
answer to have a status (the API was unreachable, or the call was never sent), so a
`ErrInvalidDocument` with a status came from the API and one without it came from here.

Neither field ever contains the API key. The key travels only in a header, and an API that
quotes it back has it removed from `Message` before it is kept. This package writes nothing to any
log.
