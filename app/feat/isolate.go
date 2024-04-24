package feat

import (
	"fmt"
	"io"
	"os"
	"path"
	"syscall"
)

func CustomChroot() error {
	temp, err := os.MkdirTemp("", "Own-Docker")
	if err != nil {
		println(err)
		return err
	}
	if err = prepareNewRoot(temp); err != nil {
		fmt.Println(err)
		return err
	}
	if err = syscall.Chroot(temp); err != nil {
		fmt.Println(err)
		return err
	}

	defer os.RemoveAll(temp)
	return nil
}

func prepareNewRoot(newRoot string) error {
	// recreate the directory hierarchy for docker-explorer inside the new root
	if err := os.MkdirAll(path.Join(newRoot, "/usr/local/bin"), 0755); err != nil {
		return err
	}
	dockerExplorer := path.Join("/usr/local/bin", "docker-explorer")
	// open the original docker-explorer to copy
	src, err := os.Open(dockerExplorer)
	if err != nil {
		return err
	}
	defer src.Close()

	// create the new docker-explorer file
	dst, err := os.Create(path.Join(newRoot, dockerExplorer))
	if err != nil {
		return err
	}
	defer dst.Close()

	// copy the original docker-explorer to the new file
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	// set permissions of new docker-explorer to allow execution
	if err := dst.Chmod(0755); err != nil {
		return err
	}

	return nil
}
