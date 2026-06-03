package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-portable-packer/internal/portable"
)

func main() {
	var stubPath string
	var folder string
	var entry string
	var outPath string
	var appName string
	var requireAdmin bool

	flag.StringVar(&stubPath, "stub", "", "path to compiled stub executable")
	flag.StringVar(&folder, "folder", "", "folder to package")
	flag.StringVar(&entry, "entry", "", "entry executable path inside folder, for example myapp.exe")
	flag.StringVar(&outPath, "out", "", "output executable path")
	flag.StringVar(&appName, "app", "", "application name used as extract directory")
	flag.BoolVar(&requireAdmin, "admin", false, "request UAC elevation before extracting and launching")
	flag.Parse()

	if err := run(stubPath, folder, entry, outPath, appName, requireAdmin); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(stubPath, folder, entry, outPath, appName string, requireAdmin bool) error {
	if stubPath == "" || folder == "" || entry == "" || outPath == "" || appName == "" {
		return fmt.Errorf("usage: packer -stub stub.exe -folder dist -entry myapp.exe -out myapp-portable.exe -app myapp")
	}

	folderAbs, err := filepath.Abs(folder)
	if err != nil {
		return err
	}
	entryRel, err := normalizeEntry(folderAbs, entry)
	if err != nil {
		return err
	}

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	stub, err := os.Open(stubPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, stub); err != nil {
		stub.Close()
		return err
	}
	if err := stub.Close(); err != nil {
		return err
	}

	metaBytes, err := portable.EncodeMetadata(portable.Metadata{
		AppName:      appName,
		Entry:        entryRel,
		CreatedAt:    time.Now().UnixMilli(),
		RequireAdmin: requireAdmin,
	})
	if err != nil {
		return err
	}
	if _, err := out.Write(metaBytes); err != nil {
		return err
	}

	beforePayload, err := out.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if err := writeTarGz(out, folderAbs); err != nil {
		return err
	}
	afterPayload, err := out.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	if err := portable.WriteFooter(out, portable.Footer{
		MetaSize:    uint64(len(metaBytes)),
		PayloadSize: uint64(afterPayload - beforePayload),
	}); err != nil {
		return err
	}

	fmt.Printf("created %s\n", outPath)
	return nil
}

func normalizeEntry(folderAbs, entry string) (string, error) {
	entryPath := entry
	if !filepath.IsAbs(entryPath) {
		entryPath = filepath.Join(folderAbs, entryPath)
	}
	entryAbs, err := filepath.Abs(entryPath)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(folderAbs, entryAbs)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("entry must be inside package folder")
	}
	if _, err := os.Stat(entryAbs); err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func writeTarGz(w io.Writer, folder string) error {
	gz := gzip.NewWriter(w)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(folder, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
