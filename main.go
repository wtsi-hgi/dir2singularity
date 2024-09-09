package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sylabs/sif/v2/pkg/sif"
)

type Paths []string

func (p *Paths) Set(path string) error {
	*p = append(*p, path)

	return nil
}

func (Paths) String() string {
	return ""
}

type Replacements [][2]string

func (r *Replacements) Set(rs string) error {
	parts := strings.Split(rs, ":")
	if len(parts) != 2 {
		return ErrInvalidReplacement
	}

	*r = append(*r, [2]string{parts[0], parts[1]})

	return nil
}

func (Replacements) String() string {
	return ""
}

func (r Replacements) Replace(path string) string {
	for _, s := range r {
		if strings.HasPrefix(path, s[0]) {
			return filepath.Join(s[1], strings.TrimPrefix(path, s[0]))
		}
	}

	return path
}

type EnvVars [][2]string

func (e *EnvVars) Set(ev string) error {
	parts := strings.Split(ev, "=")
	if len(parts) != 2 {
		return ErrInvalidEnvVar
	}

	*e = append(*e, [2]string{parts[0], parts[1]})

	return nil
}

func (EnvVars) String() string {
	return ""
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func run() error {
	var (
		paths        Paths
		envs         EnvVars
		replacements Replacements
		base         string
		output       string
		tmpDir       string
	)

	flag.StringVar(&base, "b", "", "path to base singularity image")
	flag.Var(&envs, "e", "env var to add to the singularity image (can be used multiple times)")
	flag.StringVar(&output, "o", "", "output image")
	flag.Var(&paths, "p", "path to be added to image (can be used multiple times)")
	flag.Var(&replacements, "r", "replacement prefixes, format find:replace (can be used multiple times).")
	flag.StringVar(&tmpDir, "t", os.TempDir(), "directory to temporarily place squashfs file")

	flag.Parse()

	if base == "" {
		return ErrBaseRequired
	}

	if output == "" {
		return ErrOutputRequired
	}

	if len(paths) == 0 {
		return ErrPathsRequired
	}

	tmp, err := os.MkdirTemp(tmpDir, "")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tmp)

	sqfsFile := filepath.Join(tmp, "dir.sqfs")

	sqfs, err := NewSquashFS(sqfsFile)
	if err != nil {
		return err
	}

	for _, p := range paths {
		if err := sqfs.WriteDirStructure(p, replacements.Replace(p)); err != nil {
			return err
		}
	}

	if len(envs) > 0 {
		if err := writeEnvs(sqfs, envs); err != nil {
			return err
		}
	}

	if err := sqfs.Close(); err != nil {
		return err
	}

	if err := cloneBase(base, output); err != nil {
		return err
	}

	return addToSIF(output, sqfsFile)
}

func writeEnvs(sqfs *SquashFS, envs EnvVars) error {
	var buf bytes.Buffer

	for _, env := range envs {
		fmt.Fprintf(&buf, "export %s=%q\n", env[0], env[1])
	}

	if err := sqfs.WriteDir("/.singularity.d"); err != nil {
		return err
	} else if err = sqfs.WriteDir("/.singularity.d/env"); err != nil {
		return err
	} else if err = sqfs.WriteData(buf.Bytes(), "/.singularity.d/env/99_dir2singularity.sh"); err != nil {
		return err
	}

	return nil
}

func cloneBase(base, output string) (err error) {
	b, err := os.Open(base)
	if err != nil {
		return err
	}

	defer b.Close()

	o, err := os.Create(output)
	if err != nil {
		return err
	}

	defer func() {
		if errr := o.Close(); errr != nil && err == nil {
			err = errr
		}
	}()

	_, err = io.Copy(o, b)

	return err
}

func addToSIF(baseSIF, squashfs string) error {
	f, err := os.OpenFile(baseSIF, os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	sq, err := os.Open(squashfs)
	if err != nil {
		return err
	}

	defer sq.Close()

	s, err := sif.LoadContainer(f)
	if err != nil {
		return err
	}

	di, err := sif.NewDescriptorInput(sif.DataPartition, sq, sif.OptPartitionMetadata(sif.FsSquash, sif.PartOverlay, s.PrimaryArch()))
	if err != nil {
		return err
	}

	if err := s.AddObject(di); err != nil {
		return err
	}

	return s.UnloadContainer()
}

var (
	ErrInvalidReplacement = errors.New("invalid replacement")
	ErrInvalidEnvVar      = errors.New("invalid environmental variable")
	ErrBaseRequired       = errors.New("base required")
	ErrOutputRequired     = errors.New("output required")
	ErrPathsRequired      = errors.New("at least one path must be specified")
)
