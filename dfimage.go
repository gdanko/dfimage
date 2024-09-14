package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	flags "github.com/jessevdk/go-flags"
	"golang.org/x/sys/unix"
)

const DOCKER_API_VERSION = "1.39" // This is because of a client version error
const VERSION = "0.1.1"

type Options struct {
	ImageName  string `short:"i" long:"image" description:"Specify the name of the image you want to inspect."`
	SocketPath string `short:"s" long:"socket" description:"Specify the path to the docker.sock file."`
	OutputFile string `short:"o" long:"outfile" description:"Write the output --outfile."`
	Version    func() `short:"V" long:"version" description:"Output version information and exit."`
}

func fileExists(path string) (exists bool) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func getSocket() (socketName string, err error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	socketPaths := []string{
		filepath.Join(user.HomeDir, ".rd", "docker.sock"),
		filepath.Join(user.HomeDir, ".docker", "run", "docker.sock"),
		"/var/run/docker.sock",
	}

	for _, socketPath := range socketPaths {
		if fileExists(socketPath) {
			return socketPath, nil
		}
	}

	return "", errors.New("failed to find the docker socket - use --socket to specify the path to docker.sock")
}

func pathExistsAndIsWritable(path string) (err error) {
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("the path %s does not exist - please choose another path", path)
	}
	ok := unix.Access(path, unix.W_OK)
	if ok != nil {
		return fmt.Errorf("the path %s is not writable - please choose another path", path)
	}
	return nil
}

func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func getStep(step string) string {
	if strings.Contains(step, "#(nop)") {
		stepBits := strings.Split(step, "#(nop) ")
		return stepBits[1]
	} else {
		return fmt.Sprintf("RUN %s", step)
	}
}

func processOptions(opts Options) (imageId string, socketName string, outputFile string, err error) {
	parser := flags.NewParser(&opts, flags.Default)
	parser.Usage = `--image <image_name:tag> [--socket /path/to/docker.sock]
	dfimage extracts a Dockerfile from the specified image name and prints it to STDOUT.`
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	if opts.ImageName == "" {
		return "", "", "", fmt.Errorf("missing required option --image")
	} else {
		imageId = opts.ImageName
	}

	if opts.SocketPath == "" {
		socketName, err = getSocket()
		if err != nil {
			return "", "", "", err
		}
	} else {
		socketName = opts.SocketPath
	}

	if opts.OutputFile != "" {
		var path = ""
		if strings.Contains(opts.OutputFile, "/") {
			// The option includes a path
			path = filepath.Dir(opts.OutputFile)
		} else {
			// There is no path here, we test cwd
			path, err = os.Getwd()
			if err != nil {
				return "", "", "", fmt.Errorf("unable to detect the current working directory")
			}
		}
		err = pathExistsAndIsWritable(path)
		if err != nil {
			return "", "", "", err
		}
	}
	return imageId, socketName, outputFile, nil
}

func getLayersWithImages(cli *client.Client, imageList []image.Summary) (layersWithImages map[string]string) {
	layersWithImages = make(map[string]string)
	for _, img := range imageList {
		inspect, _, err := cli.ImageInspectWithRaw(context.Background(), img.ID)
		if err != nil {
			panic(err)
		}
		layers := inspect.RootFS.Layers
		if len(layers) > 0 {
			lastLayerId := layers[len(layers)-1]
			layersWithImages[lastLayerId] = img.RepoTags[0]
		}
	}
	return layersWithImages
}

func findImageFromImageList(imageList []image.Summary, imageId string, repoTag string) (myImage image.Summary, err error) {
	var imageFound = false
	for _, img := range imageList {
		imageBits := strings.Split(img.ID, ":")
		if strings.HasPrefix(strings.ToLower(imageBits[1]), imageId) {
			myImage = img
			imageFound = true
		} else if repoTag != "" && slices.Contains(img.RepoTags, repoTag) {
			myImage = img
			imageFound = true
		}
	}

	if !imageFound {
		return myImage, fmt.Errorf("the image \"%s\" was not found - make sure you pull it first", repoTag)
	}
	return myImage, nil
}

func getFromImage(cli *client.Client, myImage image.Summary, layersWithImages map[string]string) (fromImage string) {
	// Need to return the error here
	var possibleFromImage string
	inspect, _, err := cli.ImageInspectWithRaw(context.Background(), myImage.ID)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	layers := inspect.RootFS.Layers
	if len(layers) > 0 {
		for _, layerId := range layers {
			_, ok := layersWithImages[layerId]
			if ok {
				possibleFromImage = layersWithImages[layerId]
				if possibleFromImage == myImage.RepoTags[0] {
					continue
				}
				fromImage = layersWithImages[layerId]
				break
			}
		}
	}
	return fromImage
}

func parseImageHistory(cli *client.Client, myImage image.Summary, fromImage string) (dockerCommands []string) {
	// Need to return the error here
	var fromLastCreatedBy string

	imageHistory, err := cli.ImageHistory(context.Background(), myImage.RepoTags[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if fromImage != "" {
		fromImageHistory, err := cli.ImageHistory(context.Background(), fromImage)
		if err != nil {
			panic(err)
		}
		for _, fromImageEvent := range fromImageHistory {
			fromLastCreatedBy = fromImageEvent.CreatedBy
			break
		}
	}
	for _, imageEvent := range imageHistory {
		if fromLastCreatedBy != "" && imageEvent.CreatedBy == fromLastCreatedBy {
			break
		}
		sanitizedCommand := standardizeSpaces(getStep(imageEvent.CreatedBy))
		sanitizedCommand = strings.Replace(sanitizedCommand, "/bin/sh -c ", "", -1)
		sanitizedCommand = strings.Replace(sanitizedCommand, "&&", "\n        &&", -1)
		dockerCommands = append(dockerCommands, sanitizedCommand)
	}
	return dockerCommands
}

func main() {
	var err error
	var repoTag string

	opts := Options{}

	opts.Version = func() {
		fmt.Printf("dfimage version %s\n", VERSION)
		os.Exit(0)
	}

	// Process the options
	imageId, socketName, outputFile, err := processOptions(opts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Get the image name
	if strings.Contains(imageId, ":") {
		repoTag = imageId
	} else {
		repoTag = fmt.Sprintf("%s:latest", imageId)
	}

	// Create the client
	cli, err := client.NewClientWithOpts(
		client.WithHost(fmt.Sprintf("unix://%s", socketName)),
		client.WithVersion(DOCKER_API_VERSION),
	)
	if err != nil {
		fmt.Printf("unable to create the docker client: %s\n", err)
		os.Exit(1)
	}

	// Fetch the image list
	imageList, err := cli.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		fmt.Printf("unable to generate the list of images: %s\n", err)
		os.Exit(1)
	}

	// Find the image in the list of imageList
	myImage, err := findImageFromImageList(imageList, imageId, repoTag)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Get layers with images
	layersWithImages := getLayersWithImages(cli, imageList)

	// Get the FROM image
	fromImage := getFromImage(cli, myImage, layersWithImages)

	// Parse image history
	dockerCommands := parseImageHistory(cli, myImage, fromImage)

	// Handle the FROM image
	if fromImage != "" {
		dockerCommands = append(dockerCommands, fmt.Sprintf("FROM %s", fromImage))
	} else {
		dockerCommands = append(dockerCommands, "FROM <base image not found locally>")
	}

	// Reverse the list of commands for output
	slices.Reverse(dockerCommands)

	// Print the output to either file or STDOUT
	if outputFile != "" {
		f, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer f.Close()
		for _, dockerCommand := range dockerCommands {
			_, err = f.WriteString(dockerCommand)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
		fmt.Printf("File successfully written to %s.\n", outputFile)
	} else {
		for _, dockerCommand := range dockerCommands {
			fmt.Println(dockerCommand)
		}
	}
}
