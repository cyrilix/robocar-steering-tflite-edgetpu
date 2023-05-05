package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cyrilix/robocar-steering-tflite-edgetpu/pkg/tools"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"path"
	"strconv"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

func PullOciImage(ociRef string, modelsDir string) (modelPath string, modelType tools.ModelType, imgWidth, imgHeight int, horizon int, err error) {
	repository := strings.Split(ociRef, ":")[0]
	tag := strings.Split(ociRef, ":")[1]

	manifest, err := fetchManifest(repository, tag)
	if err != nil {
		err = fmt.Errorf("unable to fetch manifest '%s': %w", ociRef, err)
		return
	}

	// 0. Create a file store
	modelStore := path.Join(modelsDir, manifest.Annotations["category"])
	fs, err := file.New(modelStore)
	if err != nil {
		return
	}
	defer fs.Close()

	// 1. Connect to a remote repository
	ctx := context.Background()
	repo, err := remote.NewRepository(repository)
	if err != nil {
		return
	}

	// 2. Copy from the remote repository to the file store
	_, err = oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions)
	if err != nil {
		return
	}
	modelType = tools.ParseModelType(manifest.Annotations["type"])
	imgWidth, err = strconv.Atoi(manifest.Annotations["img_width"])
	if err != nil {
		err = fmt.Errorf("unable to convert image width '%v' to integer: %w", manifest.Annotations["img_width"], err)
		return
	}
	imgHeight, err = strconv.Atoi(manifest.Annotations["img_height"])
	if err != nil {
		err = fmt.Errorf("unable to convert image height '%v' to integer: %w", manifest.Annotations["img_height"], err)
		return
	}
	if _, ok := manifest.Annotations["horizon"]; ok {
		horizon, err = strconv.Atoi(manifest.Annotations["horizon"])
		if err != nil {
			err = fmt.Errorf("unable to convert horizon '%v' to integer: %v", manifest.Annotations["horizon"], err)
			return
		}
	} else {
		horizon = 0
	}
	modelPath = path.Join(modelStore, manifest.Layers[0].Annotations["org.opencontainers.image.title"])
	return
}

func fetchManifest(repository string, tag string) (*v1.Manifest, error) {
	repo, err := remote.NewRepository(repository)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	descriptor, err := repo.Resolve(ctx, tag)
	if err != nil {
		panic(err)
	}
	rc, err := repo.Fetch(ctx, descriptor)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch manifest for image '%s:%s': %w", repository, tag, err)
	}
	defer rc.Close() // don't forget to close
	pulledBlob, err := content.ReadAll(rc, descriptor)
	if err != nil {
		return nil, fmt.Errorf("unable to read manifest content for image '%s:%s': %w", repository, tag, err)
	}

	var manifest v1.Manifest
	err = json.Unmarshal(pulledBlob, &manifest)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarsh json manifest content for image '%s:%s': %w", repository, tag, err)
	}
	return &manifest, nil
}
