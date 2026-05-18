package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	buildDockerfile  string
	buildTag         string
	buildNoCache     bool
	buildKeep        bool
	buildArgs        []string
	buildFormat      string
	buildQuiet       bool
	buildMemSize     int64
	buildVcpuCount   int64
	buildFsSize      int64
	buildRootfsName  string
	buildKernelName  string
)

var buildCmd = &cobra.Command{
	Use:   "build [PATH]",
	Short: "Build a Vers commit from a Dockerfile",
	Long: `Build a Vers commit by executing a Dockerfile against a throwaway VM.

Each instruction is executed in order and committed as a "layer". Layers are
cached to .vers/buildcache.json keyed by (parent commit, instruction, content
hash) so repeat builds only re-run changed steps.

FROM semantics (v1):
  FROM scratch            - start a fresh VM (requires --mem-size, --vcpu-count,
                             --fs-size-vm-mib; optional --rootfs, --kernel)
  FROM <tag>              - resolve as a vers tag first, then as a commit id
  FROM <commit-id>        - restore directly from a commit

Supported instructions:
  FROM, RUN, COPY, ADD (local only), ENV, ARG, WORKDIR, USER,
  LABEL, CMD, ENTRYPOINT, EXPOSE

Examples:
  vers build .
  vers build -f build.Dockerfile -t myapp:prod .
  vers build --no-cache --build-arg VERSION=1.2.3 .
  vers build --mem-size 2048 --vcpu-count 2 --fs-size-vm-mib 4096 .`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctxDir := "."
		if len(args) == 1 {
			ctxDir = args[0]
		}

		argMap, err := parseBuildArgs(buildArgs)
		if err != nil {
			return err
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()

		req := handlers.BuildReq{
			Dockerfile:  buildDockerfile,
			ContextDir:  ctxDir,
			Tag:         buildTag,
			NoCache:     buildNoCache,
			Keep:        buildKeep,
			BuildArgs:   argMap,
			MemSizeMib:  buildMemSize,
			VcpuCount:   buildVcpuCount,
			FsSizeVmMib: buildFsSize,
			RootfsName:  buildRootfsName,
			KernelName:  buildKernelName,
		}

		view, err := handlers.HandleBuild(apiCtx, application, req)
		if err != nil {
			return err
		}

		format := pres.ParseFormat(buildQuiet, buildFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(view)
		case pres.FormatQuiet:
			fmt.Fprintln(application.IO.Out, view.CommitID)
		default:
			pres.RenderBuild(application, view)
		}
		return nil
	},
}

func parseBuildArgs(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(pairs))
	for _, p := range pairs {
		eq := strings.IndexByte(p, '=')
		if eq < 0 {
			return nil, fmt.Errorf("--build-arg %q: expected KEY=VALUE", p)
		}
		out[p[:eq]] = p[eq+1:]
	}
	return out, nil
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&buildDockerfile, "file", "f", "", "Path to Dockerfile (default: <context>/Dockerfile)")
	buildCmd.Flags().StringVarP(&buildTag, "tag", "t", "", "Tag the resulting commit with this name")
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Ignore the layer cache")
	buildCmd.Flags().BoolVar(&buildKeep, "keep", false, "Keep the builder VM alive after the build")
	buildCmd.Flags().StringArrayVar(&buildArgs, "build-arg", nil, "Set a build-time ARG value (KEY=VALUE), repeatable")
	buildCmd.Flags().StringVar(&buildFormat, "format", "", "Output format (json)")
	buildCmd.Flags().BoolVarP(&buildQuiet, "quiet", "q", false, "Print only the final commit id")

	// FROM scratch sizing (explicit, per design)
	buildCmd.Flags().Int64Var(&buildMemSize, "mem-size", 0, "Memory in MiB (required for FROM scratch)")
	buildCmd.Flags().Int64Var(&buildVcpuCount, "vcpu-count", 0, "Number of vCPUs (required for FROM scratch)")
	buildCmd.Flags().Int64Var(&buildFsSize, "fs-size-vm-mib", 0, "Root FS size in MiB (required for FROM scratch)")
	buildCmd.Flags().StringVar(&buildRootfsName, "rootfs", "", "Base rootfs name (FROM scratch only)")
	buildCmd.Flags().StringVar(&buildKernelName, "kernel", "", "Kernel image name (FROM scratch only)")
}
