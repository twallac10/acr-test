/*
Copyright © 2024 Terry Wallace terence.wallace@gmail.com
*/
package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var logger *zap.Logger

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the ociCmd.
func Execute() {
	err := ociCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func onInitialize() {
	zapOptions := []zap.Option{
		zap.AddStacktrace(zapcore.FatalLevel),
		zap.AddCallerSkip(1),
	}
	if !verbose {
		zapOptions = append(zapOptions,
			zap.IncreaseLevel(zap.LevelEnablerFunc(func(l zapcore.Level) bool { return l != zapcore.DebugLevel })),
		)
	}
	logger, _ = zap.NewDevelopment(zapOptions...)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".go-acr" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".go-acr")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func getImage() {
	logger.Debug(fmt.Sprintf("Checking image %s", ociImage))
	//Take the image name and pull it from the registry

	if !strings.HasPrefix(ociImage, "oci://") {
		logger.Fatal("Image must be in the format oci://<domain>/<org>/<repo>")
	}

	url := strings.TrimPrefix(ociImage, "oci://")

	tag := strings.Split(url, ":")[1]

	logger.Debug(fmt.Sprintf("Tag: %s", tag))
	logger.Debug(fmt.Sprintf("URL: %s", url))

	options := []name.Option{}

	r, err := name.NewRepository(strings.Split(url, ":")[0], options...)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error parsing repository: %s", err))
	}

	logger.Debug(fmt.Sprintf("Repository: %s", r))

	tags, err := remote.List(r, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error listing tags: %s", err))
	}

	logger.Debug(fmt.Sprintf("Tags: %s", tags))

	if tag == "" {
		tag = "latest"
	}

	if !slices.Contains(tags, tag) {
		logger.Debug(fmt.Sprintf("Tag %s not found in repository", tag))
	}

	ref, err := name.ParseReference(url)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error parsing image reference: %s", err))
	}

	var revision string
	switch ref.(type) {
	case name.Tag:
		var digest gcrv1.Hash
		logger.Debug("Tagged image")

		desc, err := remote.Head(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err == nil {
			digest = desc.Digest
			logger.Debug(fmt.Sprintf("Digest from Head: %s", digest.String()))
		} else {
			gdesc, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
			if err != nil {
				logger.Error(fmt.Sprintf("Error getting image: %s", err))
			}
			digest = gdesc.Descriptor.Digest
		}

		revision = fmt.Sprintf("%s@%s", tag, digest.String())

		logger.Info(fmt.Sprintf("Pulling image %s", revision))
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error getting image: %s", err))
	}
	layers, err := img.Layers()
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error getting layers: %s", err))
	}

	var layer gcrv1.Layer

	for _, l := range layers {
		mediaType, _ := l.MediaType()
		ld, err := l.Digest()
		logger.Debug(fmt.Sprintf("Layer Digest: %s", ld.String()))
		logger.Debug(fmt.Sprintf("MediaType: %s", mediaType))
		size, _ := l.Size()
		logger.Debug(fmt.Sprintf("Size: %d", size))
		if err != nil {
			logger.Error(fmt.Sprintf("Error getting media type: %s", err))
			continue
		}
		logger.Debug(fmt.Sprintf("Layer: %s", mediaType))
	}

	layer = layers[0]

	dir, err := os.MkdirTemp("/tmp", "layer")
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error creating temp dir: %s", err))
	}

	// defer os.RemoveAll(dir)

	blob, err := layer.Compressed()
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error getting compressed layer: %s", err))
	}

	defer blob.Close()

	data, err := io.ReadAll(blob)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error reading blob: %s", err))
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s", dir, "layer.tar.gz"), data, 0644)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error writing file: %s", err))
	}

	logger.Info(fmt.Sprintf("Layer written to %s/layer.tar.gz", dir))
}
