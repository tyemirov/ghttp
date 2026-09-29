//go:build darwin

package integration

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/tyemirov/ghttp/internal/certificates"
	"github.com/tyemirov/ghttp/internal/certificates/truststore"
)

type certificateRemovalRunner struct {
	commands [][]string
	trusted  bool
}

func (runner *certificateRemovalRunner) Run(_ context.Context, command string, arguments []string) error {
	runner.commands = append(runner.commands, append([]string{command}, arguments...))
	if arguments[0] == "verify-cert" && !runner.trusted {
		return untrustedCertificateError{}
	}
	if arguments[0] == "add-trusted-cert" {
		runner.trusted = true
	}
	return nil
}

type untrustedCertificateError struct{}

func (untrustedCertificateError) Error() string { return "certificate is not trusted" }
func (untrustedCertificateError) ExitCode() int { return 1 }

func TestMacOSCertificateCleanupUsesExactCertificate(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "ghttp Development CA"}, NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	certificatePath := filepath.Join(t.TempDir(), "ca.pem")
	certificateFile, err := os.Create(certificatePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(certificateFile, &pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}); err != nil {
		t.Fatal(err)
	}
	if err := certificateFile.Close(); err != nil {
		t.Fatal(err)
	}
	keychainPath := filepath.Join(t.TempDir(), "login.keychain-db")
	runner := &certificateRemovalRunner{}
	installer, err := truststore.NewInstaller(runner, certificates.NewOperatingSystemFileSystem(), truststore.Configuration{CertificateCommonName: template.Subject.CommonName, MacOSKeychainPath: keychainPath, FirefoxProfileDirectories: []string{t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.Install(context.Background(), certificatePath); err != nil {
		t.Fatal(err)
	}
	if err := installer.Install(context.Background(), certificatePath); err != nil {
		t.Fatal(err)
	}
	installCount := 0
	for _, command := range runner.commands {
		if command[1] == "add-trusted-cert" {
			installCount++
		}
	}
	if installCount != 1 {
		t.Fatalf("expected one trust installation across repeated starts, got %d", installCount)
	}
	if err := installer.Uninstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	expected := []string{"security", "delete-certificate", "-Z", fmt.Sprintf("%X", sha256.Sum256(certificateDER)), "-t", keychainPath}
	actual := runner.commands[len(runner.commands)-1]
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("certificate cleanup must select the installed certificate and its trust settings: got %v, want %v", actual, expected)
	}
}
