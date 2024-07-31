package utils

import (
	"encoding/json"
	"os"
)

const (
	symbolsFile      = "test/symbols.json"
	symbolsFileLocal = "test/symbols-local.json"
)

type Symbols struct {
	Host string `json:"host"`
	Port int64  `json:"port"`
	TLS  bool   `json:"tls"`
}

func LoadSymbolsFile() (*Symbols, error) {
	var symFile string

	_, err := os.Stat(symbolsFile)
	if err == nil {
		symFile = symbolsFile
	} else {
		symFile = symbolsFileLocal
	}

	content, err := os.ReadFile(symFile)
	if err != nil {
		return nil, err
	}

	var sym *Symbols
	if err = json.Unmarshal(content, &sym); err != nil {
		return nil, err
	}
	return sym, nil
}
