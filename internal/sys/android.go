package sys

import (
	"bytes"
	"os/exec"
	"strings"
)

// LaunchApp starts an Android app using the am command.
// If activity is empty, it launches the default activity for the package.
func LaunchApp(pkg, activity string) error {
	var cmd *exec.Cmd
	if activity != "" {
		cmd = exec.Command("am", "start", "-n", pkg+"/"+activity)
	} else {
		// Launch default activity using package name
		cmd = exec.Command("am", "start", pkg)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	// Check stderr for error messages
	if strings.Contains(stderr.String(), "Error") {
		return &LaunchError{Message: stderr.String()}
	}

	return nil
}

// RunCommand executes a shell command, script, or binary.
// The command is run via sh -c to support pipes, redirects, etc.
func RunCommand(command string) error {
	cmd := exec.Command("sh", "-c", command)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return &LaunchError{Message: err.Error()}
	}

	// Don't wait for command to finish - run in background
	go cmd.Wait()

	return nil
}

// LaunchError represents an error during app launch.
type LaunchError struct {
	Message string
}

func (e *LaunchError) Error() string {
	return e.Message
}
