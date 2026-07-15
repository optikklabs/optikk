// Package selfupdate replaces the running optikk binary with a newer release.
//
// Trust model: authenticity comes from TLS. Assets are fetched from GitHub over
// a verified HTTPS connection (see internal/httpx), so the connection is what
// establishes that a download is genuinely GitHub's. The checksum in the release
// manifest is then verified to catch a corrupt or truncated transfer — it is an
// integrity check, not an authenticity one, since the manifest travels from the
// same origin as the archive.
//
// This deliberately does not verify a publisher signature. That would add an
// anchor independent of the download server, but the signing key would live in
// the same GitHub organisation's CI secrets, so the compromise that defeats TLS
// here largely defeats the signature too. If that calculus changes — a key held
// outside GitHub, or an enterprise requirement — signing belongs in Install,
// between the checksums fetch and the archive fetch, and must fail closed: an
// optional signature check is worth nothing, because an attacker simply omits it.
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/optikklabs/optikk/internal/httpx"
	"golang.org/x/mod/semver"
)

const (
	// repo is the GitHub repository releases are published to.
	repo = "optikklabs/optikk"

	// apiBase is the GitHub API host; releaseBase serves release assets.
	apiBase     = "https://api.github.com"
	releaseBase = "https://github.com"

	// DevVersion is the version stamped into non-release builds.
	DevVersion = "0.1.0-dev"
)

// Release is a resolved release and the asset URLs for this platform.
type Release struct {
	Tag     string // e.g. "v0.4.0"
	Version string // Tag without the leading "v"

	ArchiveURL   string
	ChecksumsURL string
	ArchiveName  string
}

// Updater resolves and installs releases.
type Updater struct {
	http *http.Client

	// apiBase and releaseBase are overridable in tests.
	apiBase     string
	releaseBase string
}

// New returns an Updater that talks to GitHub over verified TLS.
func New() *Updater {
	return &Updater{
		http:        httpx.Client(60 * time.Second),
		apiBase:     apiBase,
		releaseBase: releaseBase,
	}
}

// IsDevBuild reports whether v is a local build rather than a release.
func IsDevBuild(v string) bool { return v == DevVersion }

// Latest resolves the most recent published release.
func (u *Updater) Latest(ctx context.Context) (Release, error) {
	return u.fetchRelease(ctx, u.apiBase+"/repos/"+repo+"/releases/latest")
}

// AtTag resolves a specific release by tag (e.g. "v0.4.0"). A missing "v"
// prefix is tolerated, since that is how users say version numbers.
func (u *Updater) AtTag(ctx context.Context, tag string) (Release, error) {
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return u.fetchRelease(ctx, u.apiBase+"/repos/"+repo+"/releases/tags/"+tag)
}

// fetchRelease reads a release's tag from the GitHub API and derives its asset
// URLs. Asset names follow the .goreleaser.yaml name_template, so they are
// derived rather than searched for in the API response.
func (u *Updater) fetchRelease(ctx context.Context, url string) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.http.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("contacting github: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Release{}, fmt.Errorf("no such release (see https://github.com/%s/releases)", repo)
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return Release{}, fmt.Errorf("github rate-limited this request; retry in a few minutes")
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("github returned %s", resp.Status)
	}

	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return Release{}, fmt.Errorf("reading github response: %w", err)
	}
	if body.TagName == "" {
		return Release{}, fmt.Errorf("github returned a release with no tag")
	}
	return u.release(body.TagName), nil
}

// release derives the asset layout for a tag. Names must stay in sync with
// .goreleaser.yaml (archives.name_template and checksum.name_template) and
// with install.sh, which resolves the same files.
func (u *Updater) release(tag string) Release {
	version := strings.TrimPrefix(tag, "v")
	archive := fmt.Sprintf("optikk_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	checksums := fmt.Sprintf("optikk_%s_checksums.txt", version)
	base := fmt.Sprintf("%s/%s/releases/download/%s", u.releaseBase, repo, tag)
	return Release{
		Tag:          tag,
		Version:      version,
		ArchiveName:  archive,
		ArchiveURL:   base + "/" + archive,
		ChecksumsURL: base + "/" + checksums,
	}
}

// IsNewer reports whether the candidate version is strictly newer than current.
// Both may carry a leading "v". A dev build is always considered older, so
// `optikk update` from a local build still finds a release.
func IsNewer(current, candidate string) bool {
	if IsDevBuild(current) {
		return true
	}
	return semver.Compare(canonical(current), canonical(candidate)) < 0
}

// canonical normalises a version for semver comparison. Versions that are not
// valid semver sort as the zero version, which makes them look outdated —
// the safe direction, since it prompts an update rather than suppressing one.
func canonical(v string) string {
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return "v0.0.0"
	}
	return v
}
