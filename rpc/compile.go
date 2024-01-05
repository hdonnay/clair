//go:build tools

package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	ctx := context.Background()
	log.SetFlags(log.Lshortfile)

	d, err := os.UserCacheDir()
	if err != nil {
		log.Fatalf("unable to determine cache dir: %v", err)
	}
	checkout := filepath.Join(d, "go-capnp")

	var cmd *exec.Cmd
	fi, err := os.Stat(checkout)
	// TODO(hank) Use the module cache.
	switch {
	case errors.Is(err, nil) && fi.IsDir():
		cmd = exec.CommandContext(ctx, "git", "pull", "--quiet", "origin")
		cmd.Env = append(os.Environ(), `GIT_DIR=`+filepath.Join(checkout, `.git`))
	case errors.Is(err, fs.ErrNotExist):
		cmd = exec.CommandContext(ctx, "git", "clone", "--quiet", "https://github.com/capnproto/go-capnp", checkout)
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Printf("running %q: %v", strings.Join(cmd.Args, " "), err)
		var exeErr *exec.ExitError
		if errors.As(err, &exeErr) {
			os.Exit(exeErr.ExitCode())
		}
		os.Exit(1)
	}

	cmd = exec.CommandContext(ctx, "capnp", "compile", "-I", filepath.Join(checkout, "std"), "-ogo:internal/proto")
	cmd.Args = append(cmd.Args, os.Args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Printf("running %q: %v", strings.Join(cmd.Args, " "), err)
		var exeErr *exec.ExitError
		if errors.As(err, &exeErr) {
			os.Exit(exeErr.ExitCode())
		}
		os.Exit(1)
	}
}
