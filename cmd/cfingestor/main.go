package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"

	"github.com/uoracs/cfingestor/internal/coldfront"
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

func main() {
	// filter_usernames := []string{"adm-lcrown", "adm-marka", "adm-wwinter", "root", "swmgr"}
	// filter_projects := []string{}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
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
	})

	// slog.Info("Starting server on port 8090")
	// http.ListenAndServe(":8090", mux)

	slog.Info("Reading manifest.json")
	b, err := os.ReadFile("manifest.json")
	if err != nil {
		panic(err)
	}
	var manifest AssociationManifest
	err = json.Unmarshal(b, &manifest)
	if err != nil {
		panic(err)
	}
	slog.Info("Manifest read successfully")

	slog.Info("Creating coldfront objects")
	var users []coldfront.User
	for _, u := range manifest.Users {
		// if slices.Contains(filter_usernames, u.Username) {
		// 	continue
		// }
		users = append(users, coldfront.NewUser(u.Username, u.Firstname, u.Lastname))
	}
	var projects []coldfront.Project
	for _, p := range manifest.Projects {
		projects = append(projects, coldfront.NewProject(p.Name, p.Owner))
	}
	slog.Info("Coldfront objects created successfully")

	slog.Info("Creating associations")
	var associations []coldfront.Association
	for _, p := range manifest.Projects {
		for _, u := range p.Users {
			a := coldfront.NewAssociation(u, p.Name)
			if slices.Contains(p.Admins, u) {
                a.SetManager()
			}
			associations = append(associations, a)
		}
	}
	slog.Info("Associations created successfully")

	slog.Info("Saving users")
	userBytes, err := json.Marshal(users)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("users.json", userBytes, 0644)
	if err != nil {
		panic(err)
	}

	slog.Info("Saving projects")
	projectBytes, err := json.Marshal(projects)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("projects.json", projectBytes, 0644)
	if err != nil {
		panic(err)
	}

	slog.Info("Saving associations")
	assocBytes, err := json.Marshal(associations)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("associations.json", assocBytes, 0644)
	if err != nil {
		panic(err)
	}
}
