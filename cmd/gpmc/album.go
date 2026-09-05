package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagShareToken string
	flagIsShared   bool
)

var albumCmd = &cobra.Command{
	Use:   "album",
	Short: "Manage Google Photos albums",
	Long:  "Commands to list, view photos, comment, upload photos, share, rename, and delete Google Photos albums.",
}

var albumListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all albums",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(cmd)
		if err != nil {
			return err
		}
		c, err := clientNewFromConfig(cfg)
		if err != nil {
			return err
		}

		albums, err := c.ListAlbums()
		if err != nil {
			return err
		}

		if flagOutput == "json" {
			return json.NewEncoder(os.Stdout).Encode(albums)
		}

		fmt.Printf("%-4s %-30s %-10s %s\n", "NO", "TITLE", "ITEMS", "ALBUM KEY")
		fmt.Println("--------------------------------------------------------------------------------")
		for i, a := range albums {
			title := a.Title
			if title == "" {
				title = "(Untitled)"
			}
			fmt.Printf("%-4d %-30s %-10d %s\n", i+1, title, a.ItemCount, a.AlbumKey)
		}
		return nil
	},
}

var albumPhotosCmd = &cobra.Command{
	Use:   "photos <albumKey>",
	Short: "List photos/videos in an album",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		albumKey := args[0]
		cfg, err := loadConfig(cmd)
		if err != nil {
			return err
		}
		c, err := clientNewFromConfig(cfg)
		if err != nil {
			return err
		}

		photos, err := c.ListPhotosInAlbum(albumKey, flagShareToken)
		if err != nil {
			return err
		}

		if flagOutput == "json" {
			return json.NewEncoder(os.Stdout).Encode(photos)
		}

		fmt.Printf("Photos in album %s (Total: %d):\n", albumKey, len(photos))
		for i, p := range photos {
			fmt.Printf("  [%d] %-35s (MediaKey: %s)\n", i+1, p.FileName, p.MediaKey)
		}
		return nil
	},
}

var albumCommentCmd = &cobra.Command{
	Use:   "comment <albumKey> <text>",
	Short: "Add a comment to an album",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		albumKey := args[0]
		commentText := args[1]

		cfg, err := loadConfig(cmd)
		if err != nil {
			return err
		}
		c, err := clientNewFromConfig(cfg)
		if err != nil {
			return err
		}

		commentID, err := c.AddCommentToAlbum(albumKey, commentText, flagShareToken)
		if err != nil {
			return err
		}

		if flagOutput == "json" {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{
				"comment_id": commentID,
				"album_key":  albumKey,
			})
		}

		fmt.Printf("✅ Comment successfully added!\nComment ID: %s\n", commentID)
		return nil
	},
}

var albumUploadCmd = &cobra.Command{
	Use:   "upload <albumKey> <filePath>",
	Short: "Upload a photo and add it to an album",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		albumKey := args[0]
		filePath := args[1]

		cfg, err := loadConfig(cmd)
		if err != nil {
			return err
		}
		c, err := clientNewFromConfig(cfg)
		if err != nil {
			return err
		}

		mediaKey, err := c.UploadPhotoToAlbum(albumKey, filePath, flagShareToken)
		if err != nil {
			return err
		}

		if flagOutput == "json" {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{
				"media_key": mediaKey,
				"album_key": albumKey,
				"file":      filePath,
			})
		}

		fmt.Printf("✅ Photo successfully uploaded and added to album!\nMedia Key: %s\n", mediaKey)
		return nil
	},
}

var albumShareCmd = &cobra.Command{
	Use:   "share <albumKey>",
	Short: "Share an album and generate public share link",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		albumKey := args[0]
		cfg, err := loadConfig(cmd)
		if err != nil {
			return err
		}
		c, err := clientNewFromConfig(cfg)
		if err != nil {
			return err
		}

		res, err := c.ShareAlbum(albumKey)
		if err != nil {
			return err
		}

		if flagOutput == "json" {
			return json.NewEncoder(os.Stdout).Encode(res)
		}

		fmt.Println("✅ Album successfully shared!")
		fmt.Printf("Share URL:        %s\n", res.ShareURL)
		fmt.Printf("Shared Album Key: %s\n", res.SharedAlbumKey)
		fmt.Printf("Share Token:      %s\n", res.ShareToken)
		return nil
	},
}

var albumRenameCmd = &cobra.Command{
	Use:   "rename <albumKey> <newTitle>",
	Short: "Rename an album",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		albumKey := args[0]
		newTitle := args[1]

		cfg, err := loadConfig(cmd)
		if err != nil {
			return err
		}
		c, err := clientNewFromConfig(cfg)
		if err != nil {
			return err
		}

		if err := c.RenameAlbum(albumKey, newTitle); err != nil {
			return err
		}

		fmt.Printf("✅ Album renamed to: %s\n", newTitle)
		return nil
	},
}

var albumDeleteCmd = &cobra.Command{
	Use:   "delete <albumKey>",
	Short: "Delete an album",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		albumKey := args[0]

		cfg, err := loadConfig(cmd)
		if err != nil {
			return err
		}
		c, err := clientNewFromConfig(cfg)
		if err != nil {
			return err
		}

		if err := c.DeleteAlbum(albumKey, flagIsShared); err != nil {
			return err
		}

		fmt.Printf("✅ Album %s successfully deleted\n", albumKey)
		return nil
	},
}

func init() {
	albumCmd.PersistentFlags().StringVar(&flagShareToken, "share-token", "", "Optional share token for shared albums")
	albumDeleteCmd.Flags().BoolVar(&flagIsShared, "shared", false, "Specify if deleting a shared album")

	albumCmd.AddCommand(albumListCmd)
	albumCmd.AddCommand(albumPhotosCmd)
	albumCmd.AddCommand(albumCommentCmd)
	albumCmd.AddCommand(albumUploadCmd)
	albumCmd.AddCommand(albumShareCmd)
	albumCmd.AddCommand(albumRenameCmd)
	albumCmd.AddCommand(albumDeleteCmd)

	rootCmd.AddCommand(albumCmd)
}
