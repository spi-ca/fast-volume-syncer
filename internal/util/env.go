// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import "os"

const trustedPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

// TrustedChildEnvironment returns the minimal environment passed to privileged child processes.
func TrustedChildEnvironment(extra ...string) []string {
	env := []string{"PATH=" + trustedPath, "LC_ALL=C"}
	if value, ok := os.LookupEnv("HOME"); ok && value != "" {
		env = append(env, "HOME="+value)
	}
	return append(env, extra...)
}

// TrustedPath returns the fixed PATH used when resolving helper binaries for privileged children.
func TrustedPath() string { return trustedPath }
