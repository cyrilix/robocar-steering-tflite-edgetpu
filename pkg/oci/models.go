package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cyrilix/robocar-steering-tflite-edgetpu/pkg/tools"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.uber.org/zap"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"path"
	"strconv"
)

func PullOciImage(ctx context.Context, regName, repoName, tag, modelsDir string) (modelPath string, modelType tools.ModelType, imgWidth, imgHeight int, horizon int, err error) {

	repo, err := getRepository(ctx, regName, repoName)
	if err != nil {
		err = fmt.Errorf("unable to fetch oci artifact from '%s/%s: %w", regName, repoName, err)
		return
	}

	manifest, err := fetchManifest(ctx, repo, tag)
	if err != nil {
		err = fmt.Errorf("unable to fetch manifest '%s/%s:%s': %w", regName, repoName, tag, err)
		return
	}
	zap.S().Infof("Manifest: %v", manifest)

	// 0. Create a file store
	modelStore := path.Join(modelsDir, manifest.Annotations["category"])
	fs, err := file.New(modelStore)
	if err != nil {
		return
	}
	defer fs.Close()

	// 2. Copy from the remote repoName to the file store
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

func getRepository(ctx context.Context, registryName string, repoName string) (registry.Repository, error) {

	reg, err := remote.NewRegistry(registryName)
	if err != nil {
		return nil, fmt.Errorf("bad registry '%v': %w", registryName, err)
	}
	reg.RepositoryOptions.PlainHTTP = true

	// For debug
	//reg.Repositories(ctx, "", func(repos []string) error {
	//	for _, r := range repos {
	//		zap.S().Debugf("found repo %v", r)
	//	}
	//	return nil
	//})

	repo, err := reg.Repository(ctx, repoName)
	if err != nil {
		return nil, fmt.Errorf("unable to instanciate new repository: %w", err)
	}

	// For debug
	/*
		repo.Tags(ctx, "", func(tags []string) error {
			for _, t := range tags {
				zap.S().Debugf("found tag '%v'", t)
			}
			return nil
		})
	*/
	return repo, nil
}

func fetchManifest(ctx context.Context, repo registry.Repository, tag string) (*v1.Manifest, error) {

	descriptor, err := repo.Resolve(ctx, tag)
	zap.S().Debugf("model descriptor: %#v", descriptor)
	if err != nil {
		return nil, fmt.Errorf("unexpected error on tag resolving: %w", err)
	}
	rc, err := repo.Fetch(ctx, descriptor)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch manifest for image '%s:%s': %w", repo, tag, err)
	}
	defer rc.Close() // don't forget to close
	pulledBlob, err := content.ReadAll(rc, descriptor)
	if err != nil {
		return nil, fmt.Errorf("unable to read manifest content for image '%s:%s': %w", repo, tag, err)
	}

	var manifest v1.Manifest
	err = json.Unmarshal(pulledBlob, &manifest)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarsh json manifest content for image '%s:%s': %w", repo, tag, err)
	}
	return &manifest, nil
}
