//go:build tools

package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	debugFlag = flag.Bool("D", false, "debugging output")
	debug     = func(string, ...any) {}
)

func main() {
	ctx := context.Background()
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	if *debugFlag {
		debug = debuglog
	}

	d, err := os.UserCacheDir()
	if err != nil {
		log.Fatalf("unable to determine cache dir: %v", err)
	}
	checkout := filepath.Join(d, "go-capnp")
	debug("checkout path: %q", checkout)

	var cmd *exec.Cmd
	fi, err := os.Stat(checkout)
	// TODO(hank) Use the module cache.
	switch {
	case errors.Is(err, nil) && fi.IsDir():
		debug("path exists")
		cmd = exec.CommandContext(ctx, "git", "--git-dir", filepath.Join(checkout, `.git`), "pull", "--quiet", "origin")
	case errors.Is(err, fs.ErrNotExist):
		debug("new clone")
		cmd = exec.CommandContext(ctx, "git", "clone", "--quiet", "https://github.com/capnproto/go-capnp", checkout)
	}
	runCmd(cmd)

	cmd = exec.CommandContext(ctx, "capnp", "compile", "-I", filepath.Join(checkout, "std"), "-ogo:internal/proto")
	cmd.Args = append(cmd.Args, flag.Args()...)
	runCmd(cmd)
}

func debuglog(f string, v ...any) {
	log.Printf("DEBUG: "+f, v...)
}

func runCmd(cmd *exec.Cmd) {
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	debug("exec: %+v", cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Printf("running %q: %v", strings.Join(cmd.Args, " "), err)
		var exeErr *exec.ExitError
		if errors.As(err, &exeErr) {
			os.Exit(exeErr.ExitCode())
		}
		os.Exit(1)
	}
	debug("exec: OK")
}
