package main

import (
	"encoding/json"
	"fmt"
	"os"

	"managed-usb-hub-wails/pkg/usbtree"
)

func main() {
	fmt.Println("Enumerating USB Device Tree...")
	
	roots, err := usbtree.Enumerate()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(roots, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}
