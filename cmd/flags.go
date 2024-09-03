package cmd

import "github.com/spf13/cobra"

var (
	// Used for flags.
	ociImage string
	// ociRepo  string
	verbose bool

	ociCmd = &cobra.Command{
		Args:  cobra.OnlyValidArgs,
		Use:   "oci-repo",
		Short: "Pulls an image from an OCI registry",
		Long:  `Pulls an image from an OCI registry.`,
		Run: func(cmd *cobra.Command, args []string) {
			getImage()
		},
	}
)

func init() {
	cobra.OnInitialize(onInitialize)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	ociCmd.PersistentFlags().StringVarP(&ociImage, "ociImage", "o", "", "The image to pull from the registry")
	ociCmd.MarkPersistentFlagRequired("ociImage")
	// ociCmd.PersistentFlags().StringVarP(&ociRepo, "ociRepo", "r", "", "The repository to pull the image from")
	// ociCmd.MarkPersistentFlagRequired("ociRepo")
	ociCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

}
