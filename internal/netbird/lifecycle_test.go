package netbird

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestRewriteURLRequiresHTTPSAndEncodesOfficialURL(t *testing.T) {
	got, err := rewriteURL("https://proxy.example/fetch?url={url}", "https://github.com/a b")
	if err != nil || got == "" || got == "https://proxy.example/fetch?url=https://github.com/a b" {
		t.Fatalf("unexpected URL %q: %v", got, err)
	}
	if _, err := rewriteURL("http://proxy.example/{url}", "https://github.com"); err == nil {
		t.Fatal("accepted insecure template")
	}
	if _, err := rewriteURL("https://proxy.example/no-placeholder", "https://github.com"); err == nil {
		t.Fatal("accepted invalid template")
	}
}
func TestExtractUploadRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "bad.tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err = tw.WriteHeader(&tar.Header{Name: "../netbird", Mode: 0755, Size: 1, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write([]byte{0})
	_ = tw.Close()
	_ = gz.Close()
	_ = f.Close()
	if _, err := extractUpload(archive, "bad.tar.gz"); err == nil {
		t.Fatal("accepted path traversal")
	}
}
