package main

import (
	"fmt"

	"github.com/nginx/agent/v3/v3/internal/models"
)

func main() {
    fmt.Println(models.NewNginx("v1"))
}