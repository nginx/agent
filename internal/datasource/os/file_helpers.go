package os

import (
	"fmt"
	"os"
)

func GetPermissions(fileMode os.FileMode) string {
	return fmt.Sprintf("%#o", fileMode.Perm())
}
