package main

import (
	"archive/tar"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type SquashFS struct {
	w        *tar.Writer
	pw       io.WriteCloser
	cmd      *exec.Cmd
	existing map[string]struct{}
	now      time.Time
}

func NewSquashFS(path string) (*SquashFS, error) {
	pr, pw := io.Pipe()

	cmd := exec.Command("sqfstar", path, "-all-root")
	cmd.Stdin = pr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &SquashFS{
		w:        tar.NewWriter(pw),
		pw:       pw,
		cmd:      cmd,
		existing: make(map[string]struct{}),
		now:      time.Now(),
	}, nil
}

func (s *SquashFS) exists(path string) bool {
	if _, ok := s.existing[path]; ok {
		return true
	}

	s.existing[path] = struct{}{}

	return false
}

func (s *SquashFS) WriteDir(path string) error {
	if s.exists(path) {
		return nil
	}

	return s.w.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     path,
		Mode:     0755,
		ModTime:  s.now,
		Format:   tar.FormatGNU,
	})
}

func (s *SquashFS) WriteDirStructure(source, dest string) error {
	return fs.WalkDir(os.DirFS(source), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		fp := filepath.Join(dest, path)

		if d.Type().IsDir() {
			return s.WriteDir(fp)
		} else if d.Type().IsRegular() {
			return s.WriteFile(filepath.Join(source, path), fp)
		} else if d.Type()&fs.ModeSymlink != 0 {
			link, err := os.Readlink(filepath.Join(source, path))
			if err != nil {
				return err
			}

			if strings.HasPrefix(link, source) {
				link = filepath.Join(dest, strings.TrimPrefix(link, source))
			}

			return s.WriteSymlink(link, fp)
		}

		return nil
	})
}

func (s *SquashFS) WriteFile(source, dest string) error {
	if s.exists(dest) {
		return nil
	}

	f, err := os.Open(source)
	if err != nil {
		return err
	}

	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	perms := fi.Mode() & 0700
	perms |= perms>>3 | perms>>6
	perms &= 0755

	if err := s.w.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     dest,
		ModTime:  fi.ModTime(),
		Mode:     int64(perms),
		Size:     fi.Size(),
		Format:   tar.FormatGNU,
	}); err != nil {
		return err
	}

	if _, err := io.Copy(s.w, f); err != nil {
		return err
	}

	return nil
}

func (s *SquashFS) WriteSymlink(source, dest string) error {
	if s.exists(dest) {
		return nil
	}

	return s.w.WriteHeader(&tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     dest,
		Linkname: source,
		ModTime:  s.now,
		Format:   tar.FormatGNU,
	})
}

func (s *SquashFS) Close() error {
	if err := s.w.Close(); err != nil {
		return err
	}

	s.pw.Close()

	return s.cmd.Wait()
}
