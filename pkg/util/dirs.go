package util

import (
	"os"
	"path/filepath"
)

func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h := os.Getenv("USERPROFILE") // windows
	if h == "" {
		h = "."
	}
	return h
}

func ConfigDir() (string, error) {
	h := HomeDir()
	path := filepath.Join(h, ".jx")
	err := os.MkdirAll(path, DefaultWritePermissions)
	if err != nil {
		return "", err
	}
	return path, nil
}

func CacheDir() (string, error) {
	h, err := ConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(h, "cache")
	err = os.MkdirAll(path, DefaultWritePermissions)
	if err != nil {
		return "", err
	}
	return path, nil
}

func EnvironmentsDir() (string, error) {
	h, err := ConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(h, "environments")
	err = os.MkdirAll(path, DefaultWritePermissions)
	if err != nil {
		return "", err
	}
	return path, nil
}

func BinaryLocation() (string, error) {
	h, err := ConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(h, "bin")
	err = os.MkdirAll(path, DefaultWritePermissions)
	if err != nil {
		return "", err
	}
	return path, nil
}
