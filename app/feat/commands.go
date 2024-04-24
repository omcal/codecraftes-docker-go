package feat

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func CommandUtils(command string, args []string) error {
	command = os.Args[3]
	image := args[len(args)-1]
	args = os.Args[4:len(os.Args)]
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID,
		Chroot:     image,
	}
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(cmd.ProcessState.ExitCode())
	}
	return nil

}
