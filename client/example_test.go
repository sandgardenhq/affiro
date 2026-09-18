package client_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/client"
)

// Uploading a signed document and reading the API's verdict back.
func Example() {
	// The API key comes from the API's API keys page. Keep it out of source.
	cl, err := client.New(os.Getenv("AFFIRO_API_KEY"))
	if err != nil {
		panic(err)
	}

	// The signature comes from asig, alongside the text that was typed to produce it.
	signature, err := asig.ParseString(os.Getenv("AFFIRO_SIGNATURE"))
	if err != nil {
		panic(err)
	}
	document := strings.NewReader("the text the signature was produced from")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	analysis, err := cl.Upload(ctx, signature, document)
	switch {
	case errors.Is(err, client.ErrUnauthorized):
		fmt.Println("the API key was refused; it may have been revoked")
		return
	case errors.Is(err, client.ErrInvalidDocument):
		fmt.Println("the API would not store this document:", err)
		return
	case errors.Is(err, client.ErrUnavailable):
		fmt.Println("the API could not be reached; worth retrying:", err)
		return
	case err != nil:
		panic(err)
	}

	fmt.Println("human written:", analysis.HumanWritten)
	for _, description := range analysis.Descriptions {
		fmt.Println(" -", description)
	}

	// A later draft of the same document becomes a new version of it.
	revision := strings.NewReader("the text, revised")
	if _, err := cl.UploadVersion(ctx, analysis.DocumentID, signature, revision); err != nil {
		panic(err)
	}
}

// Pointing the client at an API other than app.affiro.com.
func ExampleWithHost() {
	host := client.DefaultHost
	if fromEnv := os.Getenv("AFFIRO_API_BASE_URL"); fromEnv != "" {
		host = fromEnv
	}
	cl, err := client.New(os.Getenv("AFFIRO_API_KEY"), client.WithHost(host))
	if err != nil {
		panic(err)
	}
	_ = cl
}
