package activedirectory

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

type AssociationManifest struct {
	Projects []Project `json:"projects"`
	Users    []User    `json:"users"`
}

type Project struct {
	Name   string   `json:"name"`
	Users  []string `json:"users"`
	Admins []string `json:"admins"`
	Owner  string   `json:"owner"`
}

type User struct {
	Username  string `json:"username"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

func SaveManifest(manifest AssociationManifest) error {
	b, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("error marshalling manifest: %v", err)
	}
	err = os.WriteFile("manifest.json", b, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}
	return nil
}

func GetCurrentHash() string {
	b, err := os.ReadFile("current.md5")
	if err != nil {
		return ""
	}
	return string(b)
}

func SetCurrentHash(h string) error {
	err := os.WriteFile("current.md5", []byte(h), 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}
	return nil
}

func ManifestPOSTHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received request")
	var manifest AssociationManifest
	var err error

	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error reading request: %v", err)
		return
	}

	err = json.Unmarshal(b, &manifest)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error parsing request: %v", err)
		return
	}

	h := r.Header.Get("Content-Hash")
	if h == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "missing Content-Hash header")
		return
	}
	ch := GetCurrentHash()
	if ch == h {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Manifest already saved")
		slog.Info("Manifest already saved")
		return
	}

	err = SaveManifest(manifest)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error saving manifest: %v", err)
		return
	}

	err = SetCurrentHash(h)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error saving hash: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Manifest saved successfully")
	slog.Info("Manifest saved successfully")
}

func ManifestGETHandler(w http.ResponseWriter, r *http.Request) {
    slog.Info("Called GET handler on activedirectory manifest endpoint")
}
