package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	deployName             string
	deployBranch           string
	deployInstallCommand   string
	deployBuildCommand     string
	deployRunCommand       string
	deployWorkingDirectory string
	deployFormat           string
	deployWait             bool
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy <owner/repo>",
	Short: "Deploy a GitHub repository",
	Long: `Deploy a GitHub repository to a new Vers project.

This creates a VM, clones the repository, installs dependencies,
builds, and runs the project. The deploy runs asynchronously —
the command returns immediately with the project and VM IDs.

Prerequisites:
  - The Vers GitHub App must be installed on the repository's organization
  - Your API key's organization must match the GitHub App installation

Examples:
  vers deploy hdresearch/my-app
  vers deploy hdresearch/my-app --branch develop
  vers deploy hdresearch/my-app --name my-project --install "npm install" --build "npm run build" --run "npm start"
  vers deploy hdresearch/my-app --working-dir packages/web
  vers deploy hdresearch/my-app --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()

		req := handlers.DeployReq{
			Repo:             args[0],
			Name:             deployName,
			Branch:           deployBranch,
			InstallCommand:   deployInstallCommand,
			BuildCommand:     deployBuildCommand,
			RunCommand:       deployRunCommand,
			WorkingDirectory: deployWorkingDirectory,
			Wait:             deployWait,
		}

		view, err := handlers.HandleDeploy(apiCtx, application, req)
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, deployFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(view)
		default:
			pres.RenderDeploy(application, view)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVar(&deployName, "name", "", "Project name (defaults to repository name)")
	deployCmd.Flags().StringVar(&deployBranch, "branch", "", "Git branch to deploy (defaults to repo's default branch)")
	deployCmd.Flags().StringVar(&deployInstallCommand, "install", "", "Install command (e.g. \"npm install\")")
	deployCmd.Flags().StringVar(&deployBuildCommand, "build", "", "Build command (e.g. \"npm run build\")")
	deployCmd.Flags().StringVar(&deployRunCommand, "run", "", "Run command (e.g. \"npm start\")")
	deployCmd.Flags().StringVar(&deployWorkingDirectory, "working-dir", "", "Working directory relative to repo root")
	deployCmd.Flags().StringVar(&deployFormat, "format", "", "Output format (json)")
	deployCmd.Flags().BoolVar(&deployWait, "wait", false, "Wait until the VM is running before returning")
}
