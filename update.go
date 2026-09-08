package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

const (
	// latestReleaseAPI reports the release marked `latest`, whose tag is the
	// short commit hash it was built from (see .github/workflows/build.yml)
	// — the same value stamped into `version`, so the two compare directly.
	latestReleaseAPI = "https://api.github.com/repos/tonybogdanov/snipp/releases/latest"

	// downloadBase resolves to the newest release's assets without having to
	// name a tag, so the installer downloaded is always the one just checked.
	downloadBase = "https://github.com/tonybogdanov/snipp/releases/latest/download/"

	// offlineText is what the user sees when GitHub can't be reached — the
	// likely cause and what to do, rather than a transport error.
	offlineText = "Couldn't reach GitHub to check for updates.\n\n" +
		"Check your internet connection and try again."
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

// errOffline wraps every failure to reach GitHub, so they all surface as
// one friendly explanation.
var errOffline = errors.New("couldn't reach GitHub")

// updateSnipp compares this build's commit hash against the latest release
// and, when they differ, downloads that release's installer and hands off
// to it. The installer is the same one used for a fresh install: it fetches
// the new binary, drops it in place, refreshes the autostart entry, stops
// the running Snipp (this process) and starts the new one — so this
// function normally doesn't return, the process is killed part-way through
// the installer's work. Reusing it, rather than swapping our own executable
// in place, keeps one code path for "get this machine onto the current
// build".
func updateSnipp() {
	latest, err := latestVersion()
	if err != nil {
		alert("Snipp", updateFailure(err))
		return
	}

	if latest == version {
		alert("Snipp", "Snipp is up to date ("+version+").\n\nNo update is available right now.")
		return
	}

	installer, err := downloadInstaller()
	if err != nil {
		alert("Snipp", updateFailure(err))
		return
	}

	if err := exec.Command(installer).Start(); err != nil {
		os.Remove(installer)
		alert("Snipp", "Snipp couldn't start the updater:\n"+err.Error())
		return
	}

	// From here the installer owns the update — it reports progress in its
	// own window and will stop this process shortly. The downloaded file is
	// deliberately left in the temp dir: it's still executing, and this
	// process won't outlive it to clean up.
}

// updateFailure turns an update error into something worth reading: the
// offline case gets the plain-language explanation, anything else names
// what went wrong.
func updateFailure(err error) string {
	if errors.Is(err, errOffline) {
		return offlineText
	}
	return "Snipp couldn't check for updates:\n" + err.Error()
}

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

// downloadInstaller fetches the latest release's installer for this platform
// into a temp file and returns its path, executable.
func downloadInstaller() (string, error) {
	body, err := get(downloadBase + installerAsset)
	if err != nil {
		return "", err
	}
	defer body.Close()

	tmp, err := os.CreateTemp("", tmpPattern)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(tmp, body)
	tmp.Close()
	if err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("%w: %v", errOffline, err)
	}

	// CreateTemp makes the file 0600; the installer has to be executable
	// (a no-op on Windows, where the .exe suffix is what matters).
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	return tmp.Name(), nil
}

// get performs a GET and returns the response body, which the caller
// closes. GitHub rejects requests without a User-Agent, so one is always
// sent.
func get(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "snipp/"+version)

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
