// Package gitinfo reads branch, dirty state and a credential-stripped remote for a working tree.
package gitinfo

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Info describes the project a session runs in.
type Info struct {
	Name   string
	Path   string
	Branch string
	Remote string
	Dirty  *bool
}

// callTimeout keeps every git call, local metadata reads only, well under the hook's time budget.
const callTimeout = 150 * time.Millisecond

// Lookup returns git metadata for cwd, or just its basename and path when it is not a git repository.
func Lookup(cwd string) Info {
	info := Info{Name: filepath.Base(cwd), Path: cwd}

	head, ok := runGit(cwd, "rev-parse", "--show-toplevel", "--abbrev-ref", "HEAD")
	if !ok || head == "" {
		return info
	}
	lines := strings.Split(head, "\n")
	toplevel := strings.TrimSpace(lines[0])
	if toplevel == "" {
		return info
	}
	if len(lines) > 1 {
		branch := strings.TrimSpace(lines[1])
		if branch != "" && branch != "HEAD" {
			info.Branch = branch
		}
	}
	info.Name = filepath.Base(toplevel)
	info.Remote = resolveRemote(cwd)

	if status, ok := runGit(cwd, "status", "--porcelain", "--untracked-files=no"); ok {
		dirty := status != ""
		info.Dirty = &dirty
	} else {
		dirty := false
		info.Dirty = &dirty
	}
	return info
}

func runGit(cwd string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

type remote struct {
	name string
	url  string
}

// resolveRemote prefers the current branch's upstream remote, else "origin", else the first configured remote.
func resolveRemote(cwd string) string {
	remotes := remoteURLs(cwd)
	if len(remotes) == 0 {
		return ""
	}

	upstreamName := ""
	if out, ok := runGit(cwd, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); ok && out != "" {
		upstreamName = strings.SplitN(out, "/", 2)[0]
	}

	chosen := ""
	switch {
	case upstreamName != "" && hasRemote(remotes, upstreamName):
		chosen = upstreamName
	case hasRemote(remotes, "origin"):
		chosen = "origin"
	default:
		chosen = remotes[0].name
	}

	for _, r := range remotes {
		if r.name == chosen {
			return NormalizeRemote(r.url)
		}
	}
	return ""
}

func hasRemote(remotes []remote, name string) bool {
	for _, r := range remotes {
		if r.name == name {
			return true
		}
	}
	return false
}

// remoteURLs lists every "remote.<name>.url" entry in the order git config reports them.
func remoteURLs(cwd string) []remote {
	out, ok := runGit(cwd, "config", "--get-regexp", `^remote\..*\.url$`)
	if !ok || out == "" {
		return nil
	}
	var result []remote
	for _, line := range strings.Split(out, "\n") {
		idx := strings.Index(line, " ")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		url := strings.TrimSpace(line[idx+1:])
		if !strings.HasPrefix(key, "remote.") || !strings.HasSuffix(key, ".url") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(key, "remote."), ".url")
		result = append(result, remote{name: name, url: url})
	}
	return result
}

// NormalizeRemote strips scheme, credentials and a ".git" suffix: "owner/repo" for github.com, "host/owner/repo" otherwise.
func NormalizeRemote(raw string) string {
	url := strings.TrimSpace(raw)
	if url == "" {
		return ""
	}
	url = strings.TrimSuffix(url, ".git")

	var host, path string
	switch {
	case strings.HasPrefix(url, "git@"):
		rest := url[len("git@"):]
		colon := strings.Index(rest, ":")
		if colon < 0 {
			return ""
		}
		host = rest[:colon]
		path = rest[colon+1:]
	case strings.Contains(url, "://"):
		idx := strings.Index(url, "://")
		rest := url[idx+3:]
		if at := strings.Index(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		slash := strings.Index(rest, "/")
		if slash < 0 {
			return ""
		}
		host = rest[:slash]
		path = rest[slash+1:]
	default:
		return ""
	}

	path = strings.Trim(path, "/")
	if host == "" || path == "" {
		return ""
	}
	if strings.EqualFold(host, "github.com") {
		return path
	}
	return host + "/" + path
}
