package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gcarrenho/routeguide2/internal/core/model"
)

func main() {
	url := "http://localhost:8080/routeguide"

	// Client https to do the request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("Error Getting feature:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: Unexpected code %d\n", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Cuerpo de la respuesta:", string(body))
		return
	}

	// Response decode
	var feature model.Feature
	if err := json.NewDecoder(resp.Body).Decode(&feature); err != nil {
		fmt.Println("Error to decode:", err)
		return
	}

	fmt.Printf("Feature : %+v\n", feature)
}
