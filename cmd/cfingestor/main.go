package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"os"
// )

// type AssociationManifest struct {
// 	Projects []Project `json:"projects"`
// 	Users    []User    `json:"users"`
// }
//
// type Project struct {
// 	Name   string   `json:"name"`
// 	Users  []string `json:"users"`
// 	Admins []string `json:"admins"`
// 	Owner  string   `json:"owner"`
// }
//
// type User struct {
// 	Username  string `json:"username"`
// 	Firstname string `json:"firstname"`
// 	Lastname  string `json:"lastname"`
// }
//
// func SaveManifest(manifest AssociationManifest) error {
// 	// Save the manifest to a file
// 	err := os.WriteFile("manifest.json", []byte(fmt.Sprintf("%v", manifest)), 0644)
// 	if err != nil {
// 		return fmt.Errorf("error writing file: %v", err)
// 	}
// 	return nil
// }

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})

	// mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Println("Received request")
	// 	var manifest AssociationManifest
	// 	var err error
	// 	if err != nil {
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		fmt.Fprintf(w, "error parsing request: %v", err)
	// 		return
	// 	}
	// 	err := SaveManifest(manifest)
	// 	if err != nil {
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		fmt.Fprintf(w, "error saving manifest: %v", err)
	// 		return
	// 	}
	// 	w.WriteHeader(http.StatusOK)
	// 	fmt.Fprintf(w, "Manifest saved successfully")
	// })

	fmt.Println("Starting server on port 8090")
	http.ListenAndServe(":8090", mux)
}
