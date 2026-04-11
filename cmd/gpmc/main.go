package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xob0t/gpmc-go/internal/client"
	"github.com/xob0t/gpmc-go/internal/config"
	"github.com/xob0t/gpmc-go/internal/web"
)

var (
	// CLI flags
	flagAuthData     string
	flagAlbum        string
	flagProxy        string
	flagProgress     bool
	flagRecursive    bool
	flagThreads      int
	flagForceUpload  bool
	flagDeleteFromHost bool
	flagUseQuota     bool
	flagSaver        bool
	flagTimeout      int
	flagLogLevel     string
	flagFilter       string
	flagExclude      bool
	flagRegex        bool
	flagIgnoreCase   bool
	flagMatchPath    bool
	flagConfig       string
	flagInitConfig   bool
	flagPort         int
)

var rootCmd = &cobra.Command{
	Use:   "gpmc <path> [flags]",
	Short: "Google Photos mobile client",
	Long: `Google Photos client based on reverse engineered mobile API.

Upload files or directories to Google Photos with support for:
- Unlimited uploads in original quality
- Album creation based on directory structure
- Concurrent uploads with configurable threads
- Hash-based duplicate detection
- File filtering and recursive directory scanning`,
	Args: cobra.MinimumNArgs(1),
	RunE: runUpload,
}

var updateCacheCmd = &cobra.Command{
	Use:   "update-cache",
	Short: "Update local library cache",
	Long:  "Incrementally update local library cache from Google Photos",
	RunE:  runUpdateCache,
}

var initConfigCmd = &cobra.Command{
	Use:   "init-config",
	Short: "Create a template config file",
	Long:  "Create a template configuration file at ~/.gpmc/config.yaml",
	RunE:  runInitConfig,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Web UI server",
	Long:  "Launch an interactive web dashboard with Tailwind CSS for uploading files and folders",
	RunE:  runServe,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Upload flags
	rootCmd.Flags().StringVar(&flagAuthData, "auth_data", "", "Google auth data (or set GP_AUTH_DATA env var)")
	rootCmd.Flags().StringVar(&flagAlbum, "album", "", "Add uploaded media to album (use \"AUTO\" for directory-based albums)")
	rootCmd.Flags().StringVar(&flagProxy, "proxy", "", "Proxy URL (format: protocol://username:password@host:port)")
	rootCmd.Flags().BoolVar(&flagProgress, "progress", false, "Display upload progress")
	rootCmd.Flags().BoolVar(&flagRecursive, "recursive", false, "Scan directories recursively")
	rootCmd.Flags().IntVar(&flagThreads, "threads", 1, "Number of concurrent upload threads")
	rootCmd.Flags().BoolVar(&flagForceUpload, "force-upload", false, "Upload files even if already present in Google Photos")
	rootCmd.Flags().BoolVar(&flagDeleteFromHost, "delete-from-host", false, "Delete files from host after successful upload")
	rootCmd.Flags().BoolVar(&flagUseQuota, "use-quota", false, "Count uploads against storage quota")
	rootCmd.Flags().BoolVar(&flagSaver, "saver", false, "Upload in storage saver quality")
	rootCmd.Flags().IntVar(&flagTimeout, "timeout", 60, "Request timeout in seconds")
	rootCmd.Flags().StringVar(&flagLogLevel, "log-level", "INFO", "Log level (DEBUG|INFO|WARNING|ERROR|CRITICAL)")
	rootCmd.Flags().StringVar(&flagFilter, "filter", "", "Filter expression for file selection")
	rootCmd.Flags().BoolVar(&flagExclude, "exclude", false, "Exclude files matching filter")
	rootCmd.Flags().BoolVar(&flagRegex, "regex", false, "Use regex for filtering")
	rootCmd.Flags().BoolVar(&flagIgnoreCase, "ignore-case", false, "Case-insensitive filtering")
	rootCmd.Flags().BoolVar(&flagMatchPath, "match-path", false, "Match against full path instead of filename")
	rootCmd.Flags().StringVar(&flagConfig, "config", "", "Config file path (default: ~/.gpmc/config.yaml)")

	// Update cache flags
	updateCacheCmd.Flags().BoolVar(&flagProgress, "progress", true, "Display cache update progress")
	updateCacheCmd.Flags().IntVar(&flagTimeout, "timeout", 60, "Request timeout in seconds")
	updateCacheCmd.Flags().StringVar(&flagLogLevel, "log-level", "INFO", "Log level (DEBUG|INFO|WARNING|ERROR|CRITICAL)")

	// Add subcommands
	rootCmd.AddCommand(updateCacheCmd)
	rootCmd.AddCommand(initConfigCmd)
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&flagPort, "port", "p", 8080, "Port to run the web server on")
}

func runUpload(cmd *cobra.Command, args []string) error {
	// Handle init-config flag first
	if flagInitConfig {
		return runInitConfig(cmd, args)
	}

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create client
	c, err := client.New(cfg.AuthData,
		client.WithTimeout(flagTimeout),
		client.WithLanguage(cfg.Language),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Build upload options
	uploadOpts := []client.UploadOption{
		client.WithAlbum(flagAlbum),
		client.WithProgress(flagProgress),
		client.WithRecursive(flagRecursive),
		client.WithThreads(flagThreads),
		client.WithForceUpload(flagForceUpload),
		client.WithDeleteFromHost(flagDeleteFromHost),
		client.WithUseQuota(flagUseQuota),
		client.WithSaver(flagSaver),
		client.WithFilter(flagFilter),
		client.WithFilterExclude(flagExclude),
		client.WithFilterRegex(flagRegex),
		client.WithFilterIgnoreCase(flagIgnoreCase),
		client.WithFilterMatchPath(flagMatchPath),
	}

	// Handle multiple paths or single path
	var target interface{} = args
	if len(args) == 1 {
		target = args[0]
	}

	// Upload
	results, err := c.Upload(target, uploadOpts...)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	// Print results
	for path, mediaKey := range results {
		fmt.Printf("%s -> %s\n", filepath.Base(path), mediaKey)
	}

	return nil
}

func runUpdateCache(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create client
	c, err := client.New(cfg.AuthData,
		client.WithTimeout(flagTimeout),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Update cache
	if err := c.UpdateCache(flagProgress, 10); err != nil {
		return fmt.Errorf("cache update failed: %w", err)
	}

	return nil
}

func runServe(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create client
	c, err := client.New(cfg.AuthData,
		client.WithTimeout(flagTimeout),
		client.WithLanguage(cfg.Language),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Create and start server
	srv := web.NewServer(c, flagPort)
	return srv.Start()
}

func runInitConfig(cmd *cobra.Command, args []string) error {
	configPath := flagConfig
	if configPath == "" {
		configPath = config.GetDefaultConfigPath()
	}

	if err := config.CreateTemplateConfig(configPath); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("Config file created at: %s\n", configPath)
	fmt.Println("Edit the file and set your auth_data, then run gpmc again.")
	return nil
}

func loadConfig() (*config.Config, error) {
	configPath := flagConfig
	if configPath == "" {
		configPath = config.GetDefaultConfigPath()
	}

	cfg, err := config.LoadOrDefault(configPath)
	if err != nil {
		return nil, err
	}

	// Merge with CLI flags
	if flagAuthData != "" {
		cfg.AuthData = flagAuthData
	}
	if flagProxy != "" {
		cfg.Proxy = flagProxy
	}
	if flagTimeout != 60 {
		cfg.Timeout = flagTimeout
	}
	if flagThreads != 1 {
		cfg.Threads = flagThreads
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
