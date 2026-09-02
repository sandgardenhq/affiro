package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/browser"
	"github.com/sandgardenhq/affiro/internal/random"
)

func Authenticate(ctx context.Context, host string) (string, error) {
	// TODO: endpoint / model definitions in a place clients can get them
	clientSlug := random.String(64)
	err := browser.OpenURL(host + "/api/v1/auth/initiate?client-slug=" + clientSlug)
	if err != nil {
		return "", err
	}
	cl2 := &http.Client{
		Timeout: 5 * time.Minute,
	}
	resp, err := cl2.Get(host + "/api/v1/auth/claim?client-slug=" + clientSlug)
	if err != nil {
		return "", err
	}
	defer func() {
		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			fmt.Println("failed to discard auth response: " + err.Error())
		}
		err = resp.Body.Close()
		if err != nil {
			fmt.Println("failed to close auth response: " + err.Error())
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth claim never completed: %v", resp.StatusCode)
	}
	type authCompleteResponse struct {
		JWT string `json:"token"`
	}
	var aResp authCompleteResponse
	err = json.NewDecoder(resp.Body).Decode(&aResp)
	if err != nil {
		return "", fmt.Errorf("failed to decode json: %w", err)
	}
	return aResp.JWT, nil
}
