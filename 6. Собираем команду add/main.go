package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	switch os.Args[1] {
	case "init":
		if len(os.Args) != 2 {
			usage()
			os.Exit(2)
		}
		if err := initRepository(root); err != nil {
			fatal(err)
		}
		fmt.Println(".gitbase initialized")
	case "add":
		if len(os.Args) < 3 {
			fatal(fmt.Errorf("please specify at least one file to add"))
		}
		if err := addFiles(root, root, os.Args[2:]); err != nil {
			fatal(err)
		}
		fmt.Printf("files added: %s\n", strings.Join(os.Args[2:], ", "))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "use: gitbase init")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error: ", err)
	os.Exit(1)
}

func initRepository(root string) error {
	gitDir := filepath.Join(root, ".gitbase")

	for _, directory := range []string{"objects", "refs/heads"} {
		if err := os.MkdirAll(filepath.Join(gitDir, directory), fs.ModePerm); err != nil {
			return fmt.Errorf("create %s: %w", directory, err)
		}
	}

	if err := writeIfMissing(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n")); err != nil {
		return err
	}

	if err := writeIfMissing(filepath.Join(gitDir, "refs", "heads", "main"), nil); err != nil {
		return err
	}

	return writeIfMissing(filepath.Join(gitDir, "index"), []byte("[]\n"))
}

func writeIfMissing(path string, data []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check %s: %w", path, err)
	}

	return os.WriteFile(path, data, fs.ModePerm)
}

func blobHash(data []byte) (string, []byte) {
	payload := append([]byte("blob\n"), data...)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), payload
}

func putBlob(root string, data []byte) (string, error) {
	hash, payload := blobHash(data)
	path := filepath.Join(root, ".gitbase", "objects", hash)

	if _, err := os.Stat(path); err == nil {
		return hash, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("check object: %w", err)
	}

	if err := os.WriteFile(path, payload, fs.ModePerm); err != nil {
		return "", err
	}

	return hash, nil
}

type indexEntry struct {
	Path     string `json:"path"`
	BlobHash string `json:"blob_hash"`
}

func readIndex(root string) ([]indexEntry, error) {
	data, err := os.ReadFile(filepath.Join(root, ".gitbase", "index"))
	if err != nil {
		return nil, fmt.Errorf("First, run gitbase init: %w", err)
	}
	var entries []indexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func saveIndex(root string, entries []indexEntry) error {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, ".gitbase", "index"), append(data, '\n'), fs.ModePerm)
}

func filePath(root, workingDirectory, argument string) (string, string, error) {
	path := argument
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDirectory, path)
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}

	relative, err := filepath.Rel(root, absolute)

	if err != nil {
		return "", "", err
	}

	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("file %q is outside the repository", argument)
	}

	const maxPathParts = 2
	parts := strings.SplitN(filepath.ToSlash(relative), "/", maxPathParts)

	firstComponent := parts[0]
	if strings.EqualFold(firstComponent, ".gitbase") {
		return "", "", fmt.Errorf("cannot add service file %q", argument)
	}

	info, err := os.Stat(absolute)
	if err != nil {
		return "", "", err
	}

	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("%q is not a regular file", argument)
	}

	return filepath.ToSlash(relative), absolute, nil
}

func addFiles(root, workingDirectory string, arguments []string) error {
	entries, err := readIndex(root)
	if err != nil {
		return err
	}

	byPath := make(map[string]string, len(entries))
	for _, entry := range entries {
		byPath[entry.Path] = entry.BlobHash
	}

	for _, argument := range arguments {
		relative, absolute, err := filePath(root, workingDirectory, argument)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(absolute)
		if err != nil {
			return err
		}

		hash, err := putBlob(root, data)
		if err != nil {
			return err
		}

		byPath[relative] = hash
	}

	entries = make([]indexEntry, 0, len(byPath))
	for path, hash := range byPath {
		entries = append(entries, indexEntry{Path: path, BlobHash: hash})
	}

	return saveIndex(root, entries)
}
