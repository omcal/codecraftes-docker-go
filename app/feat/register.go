package feat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	registryScheme    = "https"
	registry          = "registry.docker.io"
	registryHost      = "registry.hub.docker.com"
	repository        = "library"
	manifestMediaType = "application/vnd.docker.distribution.manifest.v2+json"
	imageManifestType = "application/vnd.oci.image.manifest.v1+json"
	authHostScheme    = "https"
	authHost          = "auth.docker.io"
)

func getToken(image string) (string, error) {
	url := fmt.Sprintf("%s://%s/token?service=%s&scope=repository:library/%s:pull", authHostScheme, authHost, registry, image)
	res, err := http.Get(url)
	if err != nil {
		return "the token get call has an issue", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("[getToken] Received error : %d", res.StatusCode)
	}
	var data struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return "", err
	}
	return data.Token, nil
}

type Manifest struct { //OCI format structure of a manifest

	Digest string `json:"digest"`

	MediaType string `json:"mediaType"`

	Platform struct {
		Architecture string `json:"architecture"`

		Os string `json:"os"`
	} `json:"platform"`

	Size int `json:"size"`
}

type Layer struct {
	Digest string `json:"digest"`

	MediaType string `json:"mediaType"`

	Size int `json:"size"`
}

type LayerResponse struct {
	SchemaVersion int `json:"schemaVersion"`

	MediaType string `json:"mediaType"`

	Config struct {
		MediaType string `json:"mediaType"`

		Size int `json:"size"`

		Digest string `json:"digest"`
	}

	Layers []Layer `json:"layers"`
}

type ManifestResponse struct {
	Manifests []Manifest `json:"manifests"`

	Layers []Layer `json:"layers"`
}

// docker manifest inspect ubuntu:v1  to check the manifest structure

func getManifests(token string, image string, tag string) (ManifestResponse, error) {

	url := fmt.Sprintf("%s://%s/v2/library/%s/manifests/%s", registryScheme, registryHost, image, tag)

	request, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {

		return ManifestResponse{}, err

	}

	request.Header.Set("Accept", manifestMediaType) // we need to pass this headers so that we can get the media type

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token)) //we are passing token as a header

	response, err := http.DefaultClient.Do(request)

	if err != nil {

		return ManifestResponse{}, err

	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {

		return ManifestResponse{}, fmt.Errorf("[getManifests] Received status code: %d", response.StatusCode)

	}

	manifestResponse := ManifestResponse{}

	err = json.NewDecoder(response.Body).Decode(&manifestResponse) // it filters the manifest from the JSON format

	if err != nil {

		return ManifestResponse{}, err

	}

	return manifestResponse, nil

}

func getLayers(token string, image string, tag string) (LayerResponse, error) {

	url := fmt.Sprintf("%s://%s/v2/library/%s/manifests/%s", registryScheme, registryHost, image, tag)

	request, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {

		return LayerResponse{}, err

	}

	request.Header.Set("Accept", imageManifestType)

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := http.DefaultClient.Do(request)

	if err != nil {

		return LayerResponse{}, err

	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {

		return LayerResponse{}, fmt.Errorf("[getLayers] Received status code: %d", response.StatusCode)

	}

	layerResponse := LayerResponse{}

	err = json.NewDecoder(response.Body).Decode(&layerResponse)

	if err != nil {

		return LayerResponse{}, err

	}

	return layerResponse, nil

}

func isRuntimePlatformManifest(manifest Manifest) bool { //it checks whether the image is suitable for the system architecture

	return manifest.Platform.Architecture == runtime.GOARCH && manifest.Platform.Os == runtime.GOOS

}

func getRuntimeLayerDigest(manifestResponse ManifestResponse) string {

	for _, manifest := range manifestResponse.Manifests {

		if isRuntimePlatformManifest(manifest) {

			return manifest.Digest

		}

	}

	return ""

}

func DownloadLayer(layer Layer, image string, token string, dir string) error {

	url := fmt.Sprintf("%s://%s/v2/library/%s/blobs/%s", registryScheme, registryHost, image, layer.Digest)

	request, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {

		return err

	}

	request.Header.Set("Accept", imageManifestType)

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := http.DefaultClient.Do(request)

	if err != nil {

		return err

	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {

		return fmt.Errorf("[DownloadLayer] Received error code: %d", response.StatusCode)

	}

	path := filepath.Join(dir, fmt.Sprintf("%s.tar", layer.Digest))

	file, err := os.Create(path)

	if err != nil {

		return err

	}

	_, err = io.Copy(file, response.Body)

	if err != nil {

		return err

	}

	return UnTar(dir, path)

}

func UnTar(dest string, tarfile string) error {

	cmd := exec.Command("tar", "-xvf", tarfile, "-C", dest)

	err := cmd.Run()

	if err != nil {

		return err

	}
	return os.Remove(tarfile)
}
func ParseTag(image string) (string, string) {
	i := strings.Index(image, ":")
	if i < 0 {
		return image, "latest"
	}
	return image[:i], image[i+1:]
}
func ImagePull(image string, dir string) (string, error) {
	imageName, tag := ParseTag(image)
	token, err := getToken(imageName)
	if err != nil {
		panic(err)
	}
	manifestResponse, err := getManifests(token, imageName, tag)
	if err != nil {
		panic(err)
	}
	runtimeDigest := ""
	layerResponse := LayerResponse{}

	if manifestResponse.Manifests == nil {
		layerResponse = LayerResponse{
			Layers: manifestResponse.Layers,
		}
	} else {

		runtimeDigest = getRuntimeLayerDigest(manifestResponse)
		layerResponse, err = getLayers(token, imageName, runtimeDigest)
		if err != nil {
			panic(err)
		}
	}
	imageDirectory := filepath.Join(dir, image)
	if err := os.MkdirAll(imageDirectory, 0766); err != nil && !os.IsExist(err) {
		panic(err)
	}

	for _, layer := range layerResponse.Layers {
		err = DownloadLayer(layer, imageName, token, imageDirectory)
		if err != nil {
			panic(err)
		}
	}

	return imageDirectory, nil

}
