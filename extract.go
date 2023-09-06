package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func ExtractDirectory(pullResult *PullResult, destDir string) (uint64, error) {
	var totalSize uint64
	layers := pullResult.Manifest.Layers
	fmt.Printf("Extracting ... (0/%d)\n", len(layers))

	for i, layer := range layers {
		id := strings.TrimPrefix(layer.Digest, SHA256Prefix)

		cursorUp()
		fmt.Printf("Extracting ... (%d/%d)\n", i+1, len(layers))

		srcPath := filepath.Join(pullResult.Path, id)
		size, err := extract(srcPath, destDir)
		if err != nil {
			return 0, fmt.Errorf("Extract %q: %v", id, err)
		}
		totalSize += size
	}

	cursorUp()
	fmt.Println("Extract done")

	return totalSize, nil
}

func extract(src, dest string) (uint64, error) {
	var size uint64
	file, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("Open src file: %w", err)
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return 0, fmt.Errorf("Open gzip file: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, fmt.Errorf("Read gzip file: %w", err)
		}
		if hdr == nil {
			continue
		}

		destPath := filepath.Join(dest, hdr.Name)

		if hdr.Typeflag == tar.TypeReg {
			err = ensureDir(destPath)
			if err != nil {
				return 0, fmt.Errorf("Ensure dir: %w", err)
			}
			dstFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode))
			if err != nil {
				return 0, fmt.Errorf("Open file: %w", err)
			}

			copied, err := io.Copy(dstFile, tr)
			if err != nil {
				return 0, fmt.Errorf("Copy file: %w", err)
			}
			size += uint64(copied)

			err = dstFile.Close()
			if err != nil {
				return 0, fmt.Errorf("Close file: %w", err)
			}
		}
	}

	return size, nil
}

func ensureDir(filepath string) error {
	dir := path.Dir(filepath)
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return err
			}
		}
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("Path %s is not a directory", dir)
	}
	return nil
}

func cursorUp() {
	fmt.Print("\x1b[A\x1b[K")
}
