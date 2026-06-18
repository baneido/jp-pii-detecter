package scripts_test

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	testVersion = "v1.2.3"
	testOS      = "linux"
	testArch    = "amd64"
	testAsset   = "jp-pii-detect_linux_amd64.tar.gz"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(dir)
}

func runScript(t *testing.T, script string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("sh", append([]string{script}, args...)...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	exit, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("%s %v: %v\n%s", script, args, err, out)
	}
	return string(out), exit.ExitCode()
}

func writeFakeReleaseArchive(t *testing.T, root string) string {
	t.Helper()
	releaseDir := filepath.Join(root, testVersion)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(releaseDir, testAsset)
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	body := "#!/bin/sh\necho fake-jp-pii-detect \"$@\"\n"
	if err := tw.WriteHeader(&tar.Header{
		Name: "jp-pii-detect",
		Mode: 0o755,
		Size: int64(len(body)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func distributionEnv(baseURL, installDir string) []string {
	return []string{
		"JP_PII_DETECT_VERSION=" + testVersion,
		"JP_PII_DETECT_OS=" + testOS,
		"JP_PII_DETECT_ARCH=" + testArch,
		"JP_PII_DETECT_RELEASE_BASE_URL=" + baseURL,
		"JP_PII_DETECT_INSTALL_DIR=" + installDir,
		"JP_PII_DETECT_CACHE_DIR=" + installDir,
	}
}

func TestInstallScriptPrintsReleaseAssetURL(t *testing.T) {
	out, code := runScript(t, "scripts/install.sh", distributionEnv("https://example.test/releases", t.TempDir()), "--print-url")
	if code != 0 {
		t.Fatalf("install.sh --print-url exit=%d\n%s", code, out)
	}
	want := "https://example.test/releases/" + testVersion + "/" + testAsset
	if strings.TrimSpace(out) != want {
		t.Fatalf("URL = %q, want %q", strings.TrimSpace(out), want)
	}
}

func TestInstallScriptInstallsFromReleaseArchive(t *testing.T) {
	releases := t.TempDir()
	writeFakeReleaseArchive(t, releases)
	installDir := filepath.Join(t.TempDir(), "bin")

	out, code := runScript(t, "scripts/install.sh", distributionEnv("file://"+releases, installDir))
	if code != 0 {
		t.Fatalf("install.sh exit=%d\n%s", code, out)
	}

	bin := filepath.Join(installDir, "jp-pii-detect")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("installed binary missing: %v\n%s", err, out)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("installed binary is not executable: %v", info.Mode())
	}
}

func TestPreCommitScriptInstallsAndRunsScanner(t *testing.T) {
	releases := t.TempDir()
	writeFakeReleaseArchive(t, releases)
	cacheDir := filepath.Join(t.TempDir(), "cache")

	out, code := runScript(t, "scripts/pre-commit.sh", distributionEnv("file://"+releases, cacheDir))
	if code != 0 {
		t.Fatalf("pre-commit.sh exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "fake-jp-pii-detect scan --staged") {
		t.Fatalf("pre-commit should run scanner with scan --staged, got:\n%s", out)
	}
}

func TestActionUsesPrebuiltBinaryInstaller(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, forbidden := range []string{"actions/setup-go", "go install", "go env"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("action.yml should not require Go toolchain; found %q", forbidden)
		}
	}
	if !strings.Contains(text, "scripts/install.sh") || !strings.Contains(text, "jp-pii-detect ${{ inputs.args }}") {
		t.Fatalf("action.yml should install a release binary and run it:\n%s", text)
	}
}

func TestPreCommitHookUsesScriptWrapper(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".pre-commit-hooks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"entry: scripts/pre-commit.sh",
		"language: script",
		"pass_filenames: false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf(".pre-commit-hooks.yaml missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "language: golang") {
		t.Fatalf(".pre-commit-hooks.yaml should not use language: golang")
	}
}

func TestReleaseWorkflowPublishesPrebuiltAssets(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"tags:",
		"'v*'",
		"GOOS=\"$GOOS\"",
		"GOARCH=\"$GOARCH\"",
		"jp-pii-detect_${GOOS}_${GOARCH}",
		"gh release create",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release workflow missing %q:\n%s", want, text)
		}
	}
}
