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
		fmt.Println("инициализирован .gitbase")

		hash, err := putBlob(root, []byte("hello"))
		if err != nil {
			fatal(err)
		}
		entries := []indexEntry{
			{Path: "z.txt", BlobHash: hash},
			{Path: "a.txt", BlobHash: hash},
		}
		if err := saveIndex(root, entries); err != nil {
			fatal(err)
		}
		loaded, err := readIndex(root)
		if err != nil {
			fatal(err)
		}

		for _, entry := range loaded {
			fmt.Println(entry.Path, entry.BlobHash)
		}

	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "use: gitbase init")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
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
