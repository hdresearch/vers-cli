package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hdresearch/vers-cli/internal/assets"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

var projectName string

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new vers project",
	Long:  `Initialize a new vers project with a vers.toml configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		// Create a hidden .vers directory
		versDir := ".vers"
		if err := os.MkdirAll(versDir, 0755); err != nil {
			return fmt.Errorf("error creating .vers directory: %w", err)
		}

		// Create a .gitignore file if it doesn't exist
		gitignorePath := ".gitignore"
		if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
			gitignoreContent := assets.GitIgnoreContent
			if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
				return fmt.Errorf("error creating .gitignore file: %w", err)
			}
		} 

		// Create .vers/refs directory
		refsDir := filepath.Join(versDir, "refs")
		if err := os.MkdirAll(refsDir, 0755); err != nil {
			return fmt.Errorf("error creating .vers/refs directory: %w", err)
		}

		// Create refs subdirectories
		if err := os.MkdirAll(filepath.Join(refsDir, "heads"), 0755); err != nil {
			return fmt.Errorf("error creating refs/heads directory: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(refsDir, "tags"), 0755); err != nil {
			return fmt.Errorf("error creating refs/tags directory: %w", err)
		}

		// Create .vers/objects directory
		objectsDir := filepath.Join(versDir, "objects")
		if err := os.MkdirAll(objectsDir, 0755); err != nil {
			return fmt.Errorf("error creating objects directory: %w", err)
		}

		// Create objects subdirectories (info and pack)
		if err := os.MkdirAll(filepath.Join(objectsDir, "info"), 0755); err != nil {
			return fmt.Errorf("error creating objects/info directory: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(objectsDir, "pack"), 0755); err != nil {
			return fmt.Errorf("error creating objects/pack directory: %w", err)
		}

		// Create .vers/info directory
		infoDir := filepath.Join(versDir, "info")
		if err := os.MkdirAll(infoDir, 0755); err != nil {
			return fmt.Errorf("error creating info directory: %w", err)
		}

		// Create .vers/hooks directory
		hooksDir := filepath.Join(versDir, "hooks")
		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			return fmt.Errorf("error creating hooks directory: %w", err)
		}

		// Create .vers/logs directory
		logsDir := filepath.Join(versDir, "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			return fmt.Errorf("error creating logs directory: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(logsDir, "refs", "heads"), 0755); err != nil {
			return fmt.Errorf("error creating logs/refs/heads directory: %w", err)
		}

		// Create .vers/HEAD file
		headFile := filepath.Join(versDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
			return fmt.Errorf("error creating .vers/HEAD file: %w", err)
		}

		// Create .vers/config file
		configFile := filepath.Join(versDir, "config")
		defaultConfig := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[vers]
	version = 1
`
		if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("error creating config file: %w", err)
		}

		// Create .vers/description file
		descFile := filepath.Join(versDir, "description")
		if err := os.WriteFile(descFile, []byte("Unnamed repository; edit this file to name the repository.\n"), 0644); err != nil {
			return fmt.Errorf("error creating description file: %w", err)
		}

		// Create an empty index file
		indexFile := filepath.Join(versDir, "index")
		if err := os.WriteFile(indexFile, []byte{}, 0644); err != nil {
			return fmt.Errorf("error creating index file: %w", err)
		}

		logoStyle := styles.AppStyle.Foreground(styles.TerminalMagenta)
		// fmt.Printf(logoStyle.Render(`		                                                                                                                                                                                                         
                                                                       
        //     ↑↑↑↑                        
        //     ↑↑↑↑↑↑↑                     
        //        ↑↑↑↑↑↑↑                  
        //           ↑↑↑↑↑                 
        //            ↑↑↑↑                 
        //            ↑↑↑↑                 
        //   ↑↑↑↑↑    ↑↑↑↑                 
        //  ↑↑↑↑↑     ↑↑↑↑                 
        //  ↑↑↑↑       ↑↑↑                 
        //  ↑↑↑↑                           
        //  ↑↑↑↑                           
        //  ↑↑↑↑                           
        //  ↑↑↑↑↑           ↑↑↑↑           
        //    ↑↑↑↑↑↑     ↑↑↑↑↑↑            
        //      ↑↑↑↑↑↑↑↑↑↑↑↑↑              
        //         ↑↑↑↑↑↑↑                 
                                                                                                                                
		// `))
		fmt.Println(logoStyle.Render(`	
		▗▖  ▗▖▗▄▄▄▖▗▄▄▖  ▗▄▄▖
		▐▌  ▐▌▐▌   ▐▌ ▐▌▐▌   
		▐▌  ▐▌▐▛▀▀▘▐▛▀▚▖ ▝▀▚▖
		 ▝▚▞▘ ▐▙▄▄▖▐▌ ▐▌▗▄▄▞▘						 
   `))

   		fmt.Printf(styles.MutedTextStyle.Render("Initialized vers repository in %s directory\n"), versDir)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Define flags for the init command
	initCmd.Flags().StringVarP(&projectName, "name", "n", "", "Project name (defaults to directory name)")
}



