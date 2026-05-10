// Package gh wraps the GitHub CLI (`gh`) for git-switch's GitHub-only flows:
// authenticating, uploading SSH/GPG keys, and switching the active gh account.
package gh

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DefaultScopes is the minimum OAuth scope set git-switch needs.
var DefaultScopes = []string{
	"admin:public_key",
	"admin:gpg_key",
	"read:user",
	"user:email",
}

// ErrNotAuthenticated indicates that no gh session exists for the requested user.
// Callers should run Login() and retry.
var ErrNotAuthenticated = errors.New("gh: not authenticated for this user")

// SSHKey is a key as returned by the GitHub API.
type SSHKey struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Key   string `json:"key"`
}

// GPGKey is a GPG key as returned by the GitHub API. The KeyID field is the
// short hex id GitHub uses (matches the `gpg --keyid-format LONG` output for
// the primary key).
type GPGKey struct {
	ID    int    `json:"id"`
	KeyID string `json:"key_id"`
}

// BindResult captures the identity gh resolved for the freshly-logged-in user.
type BindResult struct {
	Login string
	Name  string
	Email string
}

// Login runs an interactive `gh auth login` for github.com using SSH protocol
// and the requested scopes. Stdin/stdout/stderr are inherited so the user sees
// the device-code prompt and can paste it into their browser.
func Login(scopes []string) error {
	if err := EnsureInstalled(); err != nil {
		return err
	}
	args := []string{
		"auth", "login",
		"--hostname", "github.com",
		"--git-protocol", "ssh",
		"--skip-ssh-key", // git-switch handles SSH key upload itself with a meaningful title
		"--web",
	}
	if len(scopes) > 0 {
		args = append(args, "--scopes", strings.Join(scopes, ","))
	}
	cmd := exec.Command(ghBinary(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh auth login failed: %w", err)
	}
	return nil
}

// WhoAmI returns the active gh user's login, display name and primary email.
// Falls back to the user/emails endpoint when the public profile email is null
// (common when users have email privacy turned on).
func WhoAmI() (login, name, email string, err error) {
	out, err := run("api", "user")
	if err != nil {
		return "", "", "", err
	}
	var u struct {
		Login string  `json:"login"`
		Name  string  `json:"name"`
		Email *string `json:"email"`
	}
	if jerr := json.Unmarshal(out, &u); jerr != nil {
		return "", "", "", fmt.Errorf("parsing gh api user: %w", jerr)
	}
	login = u.Login
	name = u.Name
	if u.Email != nil {
		email = *u.Email
	}
	if email == "" {
		email = primaryEmail()
	}
	return login, name, email, nil
}

// primaryEmail asks the /user/emails endpoint for the verified primary address.
// Returns empty if anything goes wrong; callers should fall back to a prompt.
func primaryEmail() string {
	out, err := run("api", "user/emails")
	if err != nil {
		return ""
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if json.Unmarshal(out, &emails) != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	return ""
}

// SwitchAccount makes the given login the active gh user for github.com.
// Returns ErrNotAuthenticated if that login isn't logged in locally.
func SwitchAccount(login string) error {
	if !isAuthenticated(login) {
		return ErrNotAuthenticated
	}
	if _, err := run("auth", "switch", "-h", "github.com", "-u", login); err != nil {
		return fmt.Errorf("gh auth switch failed: %w", err)
	}
	return nil
}

// AddSSHKey uploads pubKeyPath under the given title. Idempotent — an already
// uploaded key is treated as success.
func AddSSHKey(pubKeyPath, title string) error {
	out, err := run("ssh-key", "add", pubKeyPath, "--title", title)
	if err == nil {
		return nil
	}
	if alreadyUploaded(string(out), err) {
		return nil
	}
	return fmt.Errorf("gh ssh-key add failed: %s", combinedErr(out, err))
}

// AddGPGKey uploads an ASCII-armored public key via stdin. Idempotent.
func AddGPGKey(armoredPubKey string) error {
	cmd := exec.Command(ghBinary(), "gpg-key", "add", "-")
	cmd.Stdin = strings.NewReader(armoredPubKey)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if alreadyUploaded(string(out), err) {
		return nil
	}
	return fmt.Errorf("gh gpg-key add failed: %s", combinedErr(out, err))
}

// ListSSHKeys returns the SSH keys uploaded to the active gh user's account.
func ListSSHKeys() ([]SSHKey, error) {
	out, err := run("api", "user/keys")
	if err != nil {
		return nil, err
	}
	var keys []SSHKey
	if jerr := json.Unmarshal(out, &keys); jerr != nil {
		return nil, fmt.Errorf("parsing user/keys: %w", jerr)
	}
	return keys, nil
}

// ListGPGKeys returns the GPG keys uploaded to the active gh user's account.
func ListGPGKeys() ([]GPGKey, error) {
	out, err := run("api", "user/gpg_keys")
	if err != nil {
		return nil, err
	}
	var keys []GPGKey
	if jerr := json.Unmarshal(out, &keys); jerr != nil {
		return nil, fmt.Errorf("parsing user/gpg_keys: %w", jerr)
	}
	return keys, nil
}

// DeleteSSHKey removes the SSH key with the given GitHub-side ID.
func DeleteSSHKey(id int) error {
	_, err := run("api", "-X", "DELETE", fmt.Sprintf("user/keys/%d", id))
	return err
}

// DeleteGPGKey removes the GPG key with the given GitHub-side ID.
func DeleteGPGKey(id int) error {
	_, err := run("api", "-X", "DELETE", fmt.Sprintf("user/gpg_keys/%d", id))
	return err
}

// MatchSSHKeyByContent finds an uploaded key whose body matches the given
// public-key content (the `ssh-ed25519 AAAA... comment` form). Returns nil when
// no match is found. Comparison ignores the trailing comment field, since
// GitHub strips it on upload.
func MatchSSHKeyByContent(keys []SSHKey, pub string) *SSHKey {
	target := keyBody(pub)
	if target == "" {
		return nil
	}
	for i := range keys {
		if keyBody(keys[i].Key) == target {
			return &keys[i]
		}
	}
	return nil
}

// MatchGPGKeyByID finds an uploaded GPG key whose KeyID matches. GitHub returns
// the long-form id without the 0x prefix; we trim either way.
func MatchGPGKeyByID(keys []GPGKey, keyID string) *GPGKey {
	target := strings.TrimPrefix(strings.ToUpper(keyID), "0x")
	for i := range keys {
		if strings.ToUpper(keys[i].KeyID) == target {
			return &keys[i]
		}
	}
	return nil
}

// BindAccount runs the standard "install gh, log in, fetch identity" sequence.
// Used by `add`, the migration path in runSwitch, and `reset`.
func BindAccount() (*BindResult, error) {
	if err := EnsureInstalled(); err != nil {
		return nil, err
	}
	if err := Login(DefaultScopes); err != nil {
		return nil, err
	}
	login, name, email, err := WhoAmI()
	if err != nil {
		return nil, err
	}
	return &BindResult{Login: login, Name: name, Email: email}, nil
}

// run executes `gh` with the given args, capturing stdout. Stderr is folded
// into the returned error on failure.
func run(args ...string) ([]byte, error) {
	cmd := exec.Command(ghBinary(), args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("gh %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// isAuthenticated reports whether `gh auth status` lists this user as logged in.
func isAuthenticated(login string) bool {
	cmd := exec.Command(ghBinary(), "auth", "status", "-h", "github.com")
	out, _ := cmd.CombinedOutput()
	return strings.Contains(string(out), "account "+login) ||
		strings.Contains(string(out), "as "+login) ||
		strings.Contains(string(out), "Logged in to github.com account "+login)
}

// alreadyUploaded recognises the various phrasings GitHub returns when a key
// is already on the account, so callers can treat the "error" as success.
// GPG returns "key was not added because one or more subkeys already exist:"
// (HTTP 422) which we want to treat as no-op success.
func alreadyUploaded(out string, err error) bool {
	lower := strings.ToLower(out + " " + err.Error())
	return strings.Contains(lower, "key is already in use") ||
		strings.Contains(lower, "key already exists") ||
		strings.Contains(lower, "already in use") ||
		strings.Contains(lower, "key_id already exists") ||
		strings.Contains(lower, "subkeys already exist") ||
		strings.Contains(lower, "key was not added")
}

func combinedErr(out []byte, err error) string {
	s := strings.TrimSpace(string(out))
	if s == "" {
		return err.Error()
	}
	return s
}

// keyBody returns just the algorithm + body portion of a public key, dropping
// the trailing comment if present. Used by MatchSSHKeyByContent so a renamed
// comment doesn't break matching.
func keyBody(pub string) string {
	fields := strings.Fields(strings.TrimSpace(pub))
	if len(fields) < 2 {
		return ""
	}
	return fields[0] + " " + fields[1]
}
