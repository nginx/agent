package sysutils

import (
	"fmt"
	"io"
	"os"
)

func CopyFile(src, dest string) (func() error, error) {
	in, err := os.Open(src)
	if err != nil {
		return nil, fmt.Errorf("failed creating test agent config (%s) when opening file - %v", src, err)
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return nil, fmt.Errorf("failed creating test agent config (%s) when creating file - %v", dest, err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return nil, fmt.Errorf("failed creating test agent config (%s) when copying over file - %v", src, err)
	}

	err = out.Close()
	if err != nil {
		return nil, fmt.Errorf("failed creating test agent config (%s) when closing file - %v", dest, err)
	}

	deleteFunc := func() error {
		err := os.Remove(dest)
		if err != nil {
			return fmt.Errorf("failed to delete test agent config (%s) - %v", dest, err)
		}
		return nil
	}

	return deleteFunc, nil
}
