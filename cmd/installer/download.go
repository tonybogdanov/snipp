package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// latestReleaseAPI reports the release marked `latest`, whose tag is the
	// short commit hash it was built from (see .github/workflows/build.yml).
	// The tag of the release a binary was installed from is recorded next to
	// it, so a re-run of the installer can tell "already current" from
	// "needs updating" without downloading anything.
	latestReleaseAPI = "https://api.github.com/repos/tonybogdanov/snipp/releases/latest"

	// downloadBase resolves to the newest release's assets without naming a
	// tag. The app binary is fetched from there at install time rather than
	// embedded in this installer, so an installer downloaded months ago
	// still installs the current build — the price is that installing
	// requires network access.
	downloadBase = "https://github.com/tonybogdanov/snipp/releases/latest/download/"
)

var httpClient = &http.Client{Timeout: 120 * time.Second}

// errOffline wraps every failure to reach GitHub, so the UI can show one
// friendly explanation instead of a raw transport error.
var errOffline = errors.New("couldn't reach GitHub")

// latestVersion returns the tag of the latest release — the short commit
// hash it was built from.
func latestVersion() (string, error) {
	body, err := get(latestReleaseAPI)
	if err != nil {
		return "", err
	}
	defer body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(body).Decode(&release); err != nil {
		return "", fmt.Errorf("%w: %v", errOffline, err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("%w: no released version found", errOffline)
	}
	return release.TagName, nil
}

// fetchBinary downloads the latest released Snipp binary for this platform.
func fetchBinary() ([]byte, error) {
	body, err := get(downloadBase + binaryAsset)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errOffline, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: the download was empty", errOffline)
	}
	return data, nil
}

// get performs a GET and returns the response body, which the caller
// closes. GitHub rejects requests without a User-Agent, so one is always
// sent.
func get(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "snipp-installer")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errOffline, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%w: %s", errOffline, resp.Status)
	}
	return resp.Body, nil
}
