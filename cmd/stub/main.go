package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go-portable-packer/internal/portable"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err := os.Open(self)
	if err != nil {
		return err
	}
	defer exe.Close()

	info, err := exe.Stat()
	if err != nil {
		return err
	}
	footer, err := portable.ReadFooter(exe, info.Size())
	if err != nil {
		return err
	}

	metaOffset := info.Size() - portable.FooterSize - int64(footer.PayloadSize) - int64(footer.MetaSize)
	payloadOffset := metaOffset + int64(footer.MetaSize)
	if metaOffset < 0 || payloadOffset < 0 {
		return fmt.Errorf("invalid portable payload offsets")
	}

	metaBytes := make([]byte, footer.MetaSize)
	if _, err := exe.ReadAt(metaBytes, metaOffset); err != nil {
		return err
	}
	meta, err := portable.DecodeMetadata(metaBytes)
	if err != nil {
		return err
	}

	args := os.Args[1:]
	if meta.RequireAdmin {
		relaunched, err := ensureElevated()
		if err != nil {
			return err
		}
		if relaunched {
			return nil
		}
	}

	payloadBytes := make([]byte, footer.PayloadSize)
	if _, err := exe.ReadAt(payloadBytes, payloadOffset); err != nil {
		return err
	}

	extractDir, err := appExtractDir(meta.AppName)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(extractDir); err != nil {
		return err
	}
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}
	if err := extractTarGz(bytes.NewReader(payloadBytes), extractDir); err != nil {
		return err
	}

	entry := filepath.Join(extractDir, filepath.FromSlash(meta.Entry))
	cmd := exec.Command(entry, args...)
	cmd.Dir = filepath.Dir(entry)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

func appExtractDir(appName string) (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, appName), nil
}

func extractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(dest, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
}

func safeJoin(root, name string) (string, error) {
	cleanName := filepath.Clean(filepath.FromSlash(name))
	if cleanName == "." || strings.HasPrefix(cleanName, ".."+string(os.PathSeparator)) || filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("unsafe archive path: %s", name)
	}
	target := filepath.Join(root, cleanName)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("unsafe archive path: %s", name)
	}
	return target, nil
}
