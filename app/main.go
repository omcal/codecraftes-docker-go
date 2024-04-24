package main

import (
	"fmt"
	"github.com/codecrafters-io/docker-starter-go/app/feat"
	"os"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	image := os.Args[2]
	imageDir := fmt.Sprintf("./images/%s", image)
	args := os.Args
	args = append(args, imageDir)
	if _, err := os.Stat(imageDir); err != nil {
		if os.IsNotExist(err) {
			imageDir, err = feat.ImagePull(image, "./images")
			if err != nil {
				print(err)
				os.Exit(1)
			}
		}
	}
	if err := feat.CustomChroot(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err := feat.CommandUtils(args[3], args[4:len(args)])
	if err != nil {

		fmt.Println(err)
	}
}
