package internal

import (
	"bytes"
	"os"
	"os/exec"
)

func createTempDir(tempDir string, name string) (string, error) {
	dir, err := os.MkdirTemp(tempDir, "shortcut-*")

	if err != nil {
		return "", err
	}

	return dir, nil
}

func saveShortcut(filePath string, shortcut string) error {

	err := os.WriteFile(filePath, []byte(shortcut), 0644)

	return err

}

func signShortcut(input string, output string) ([]byte, error) {

	var sign = exec.Command(
		"shortcuts",
		"sign",
		"-i", input,
		"-o", output,
		"-m", "anyone",
	)
	var stdErr bytes.Buffer
	sign.Stderr = &stdErr
	var err = sign.Run()
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(output)
	if err != nil {
		return nil, err
	}

	return content, nil
}
