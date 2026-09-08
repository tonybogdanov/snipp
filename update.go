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

// latestReleaseAPI reports the release marked `latest`: its tag is the short
// commit hash it was built from — the same value stamped into `version`, so
// the two compare directly — and its asset list carries the download URL
// for each platform's installer.
//
// The installer is resolved from that response rather than through the
// /releases/latest/download/<name> alias, which lags behind it: for a
// window after a release is published the API already reports the new tag
// while the alias still 404s, which looked exactly like being offline.
const latestReleaseAPI = "https://api.github.com/repos/tonybogdanov/snipp/releases/latest"

var httpClient = &http.Client{Timeout: 60 * time.Second}

var (
	// errOffline is a failure to reach GitHub at all — no route, no DNS, no
	// TLS. The user is offline, or something between them and GitHub is.
	errOffline = errors.New("couldn't reach GitHub")

	// errServer is reaching GitHub and being told no: a status code, a
	// response that doesn't parse, a release missing its assets. Nothing
	// the user's network can explain, so it must not be reported as if it
	// were.
	errServer = errors.New("GitHub couldn't provide the download")
)

// release is the latest release: the tag it's named after, and the download
// URL of each asset attached to it, by asset name.
type release struct {
	tag    string
	assets map[string]string
}

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
	rel, err := latestRelease()
	if err != nil {
		alert("Snipp", failureMessage(err))
		return
	}

	if rel.tag == version {
		alert("Snipp", "Snipp is up to date ("+version+").\n\nNo update is available right now.")
		return
	}

	installer, err := downloadInstaller(rel)
	if err != nil {
		alert("Snipp", failureMessage(err))
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

// failureMessage explains a failed update in terms of what it means for the
// user. Telling someone with a working connection to check their internet
// sends them looking in the wrong place, so the two causes read
// differently.
func failureMessage(err error) string {
	switch {
	case errors.Is(err, errOffline):
		return "Couldn't reach GitHub to check for updates.\n\n" +
			"Check your internet connection and try again."
	case errors.Is(err, errServer):
		return "GitHub couldn't provide the update right now.\n" +
			"If a new Snipp release just went out, give it a minute and try again.\n\n" +
			"(" + err.Error() + ")"
	default:
		return "Snipp couldn't check for updates:\n" + err.Error()
	}
}

// latestRelease fetches the latest release's tag and asset URLs.
func latestRelease() (release, error) {
	body, err := get(latestReleaseAPI)
	if err != nil {
		return release{}, err
	}
	defer body.Close()

	var parsed struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(body).Decode(&parsed); err != nil {
		return release{}, fmt.Errorf("%w: %v", errServer, err)
	}
	if parsed.TagName == "" {
		return release{}, fmt.Errorf("%w: no released version found", errServer)
	}

	rel := release{tag: parsed.TagName, assets: map[string]string{}}
	for _, asset := range parsed.Assets {
		rel.assets[asset.Name] = asset.URL
	}
	return rel, nil
}

// downloadInstaller fetches the given release's installer for this platform
// into a temp file and returns its path, executable.
func downloadInstaller(rel release) (string, error) {
	url, ok := rel.assets[installerAsset]
	if !ok {
		return "", fmt.Errorf("%w: release %s has no %s", errServer, rel.tag, installerAsset)
	}

	body, err := get(url)
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
// sent. Transport failures and 5xx are retried briefly — both are usually
// a blip, and a second attempt is cheaper than making the user click the
// menu entry again.
func get(url string) (io.ReadCloser, error) {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		var body io.ReadCloser
		var retry bool
		body, retry, err = attemptGet(url)
		if err == nil {
			return body, nil
		}
		if !retry {
			return nil, err
		}
	}
	return nil, err
}

// attemptGet makes one request, reporting whether the failure is the kind
// worth retrying.
func attemptGet(url string) (body io.ReadCloser, retry bool, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", "snipp/"+version)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("%w: %v", errOffline, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, resp.StatusCode >= 500, fmt.Errorf("%w: %s", errServer, resp.Status)
	}
	return resp.Body, false, nil
}
