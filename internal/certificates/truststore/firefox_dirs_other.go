//go:build !linux && !darwin && !windows

package truststore

func defaultFirefoxProfileDirectories() []string {
	return nil
}
