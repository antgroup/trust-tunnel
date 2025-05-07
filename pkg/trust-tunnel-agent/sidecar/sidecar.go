// Copyright The TrustTunnel Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sidecar

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"trust-tunnel/pkg/common/logutil"

	"github.com/docker/docker/api/types/container"
	imageTypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

var logger = logutil.GetLogger("trust-tunnel-agent")

const (
	defaultSidecarImage             = "trust-tunnel-sidecar:latest"
	defaultCleanLegacySidecarPeriod = 5 * time.Minute
)

type Config struct {
	// Image specifies the image of the sidecar container.
	Image string

	// ImageHubAuth specifies the authentication information for the image hub.
	ImageHubAuth string

	// Limit specifies the maximum number of sidecar containers that can be existed at the same time.
	Limit int
}

// PullMissingImage tries to pull a Docker image if it does not exist locally or force updating is true.
// It first checks if the image exists locally, then pulls the image from the registry if necessary.
func PullMissingImage(image, auth string, force bool, apiClient client.CommonAPIClient) (string, error) {
	if apiClient == nil {
		return "", fmt.Errorf("container client is not ready")
	}

	if strings.TrimSpace(image) == "" {
		image = defaultSidecarImage
	}

	exists, err := imageExists(apiClient, image)
	if err != nil {
		logger.Errorf("check image existence error: %s", err.Error())

		return "", err
	}

	if exists && !force {
		// Image exists, and we don't force the image updating, return directly.
		return image, nil
	}

	// Image not exists, or force updating is true.
	nameAndTags := strings.Split(image, ":")
	name := nameAndTags[0]
	tag := "latest"

	if len(nameAndTags) > 1 {
		tag = nameAndTags[1]
	}

	logger.Infof("pulling image %s with tag %s", name, tag)

	body, err := apiClient.ImagePull(context.Background(), name+":"+tag, imageTypes.PullOptions{RegistryAuth: base64.URLEncoding.EncodeToString([]byte(auth))})
	if err != nil {
		return "", err
	}
	defer body.Close()

	br := bufio.NewReader(body)

	for {
		line, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return "", fmt.Errorf("failed to read image pulling content: %w", err)
		}

		logger.Debugf("%s", string(line))
	}

	// Check again.
	_, _, err = apiClient.ImageInspectWithRaw(context.Background(), image)
	if err == nil {
		logger.Infof("image %s is pulled", image)

		return image, nil
	}

	return "", fmt.Errorf("failed to pull image %s", image)
}

// Init sets up the sidecar container environment.
// It primarily verifies the availability of the Docker endpoint and pulls the required sidecar image.
// If the Docker environment is not ready or the image pull fails, returns an error.
func Init(endpoint, image, auth string, apiClient client.CommonAPIClient) (string, error) {
	if apiClient == nil {
		return "", fmt.Errorf("container client is nil")
	}

	if _, err := os.Stat(strings.TrimPrefix(endpoint, "unix://")); err != nil {
		logger.Infof("docker endpoint(%v) not exits,maybe docker env not ready,ignore", strings.TrimPrefix(endpoint, "unix://"))

		return "", err
	}

	image, err := PullMissingImage(image, auth, false, apiClient)
	if err != nil {
		logger.Errorf("pull sidecar image %s failed: %v", image, err)

		return "", err
	}

	return image, nil
}

// CleanLegacyContainerPeriodically list all the containers,include the not running containers,
// and kill the container with the image of $DefaultSidecar which is not running and created an hour ago.
// In some situations, when creating a large number of sidecar sessions,
// sidecar containers may not be successfully reclaimed due to container performance issues，
// we need to clean legacy sidecar(not running and created an hour ago) container periodically.
func CleanLegacyContainerPeriodically(apiClient client.CommonAPIClient) {
	logger.Infof("start clean legacy trust-tunnel-sidecar containers  periodcally")

	if apiClient == nil {
		return
	}

	for {
		time.Sleep(defaultCleanLegacySidecarPeriod)

		containers, err := apiClient.ContainerList(context.Background(), container.ListOptions{All: true})
		if err != nil {
			logger.Errorf("failed to list containers %v", err)

			continue
		}

		var legacySidecarNum int

		for _, c := range containers {
			createdTime := time.Unix(c.Created, 0)

			if strings.HasPrefix(c.Image, defaultSidecarImage) && c.State != "running" && createdTime.Before(time.Now().Add(-time.Hour)) {
				legacySidecarNum++

				err := apiClient.ContainerRemove(context.Background(), c.ID, container.RemoveOptions{Force: true})
				if err != nil {
					logger.Errorf("remove legacy container %s error:%v", c.ID, err)

					continue
				}

				logger.Infof("remove legacy container with image %s done", c.Image)
			}
		}
	}
}

func imageExists(cli client.CommonAPIClient, image string) (bool, error) {
	_, _, err := cli.ImageInspectWithRaw(context.Background(), image)
	if err == nil {
		return true, nil
	} else if client.IsErrNotFound(err) {
		return false, nil
	} else {
		return false, err
	}
}
