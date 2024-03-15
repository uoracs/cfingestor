package coldfront

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"

	"github.com/uoracs/cfingestor/internal/activedirectory"
	"github.com/uoracs/cfingestor/internal/ingest"
)

var coldfrontCommand = "/srv/coldfront/venv/bin/coldfront"

func ingestDir(f string) string {
	return fmt.Sprintf("%s/%s", ingest.IngestDirectory, f)
}

func IngestPOSTHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("POST /ingest")
	found := CheckIngestFlag()
	if found {
		slog.Info("Ingest in progress")
		w.WriteHeader(http.StatusProcessing)
		return
	}
	StartIngestFlag()
	err := IngestManifest()
	if err != nil {
		slog.Error("Error ingesting manifest: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	EndIngestFlag()
	w.WriteHeader(http.StatusProcessing)
}

func IngestGETHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("GET /ingest")
	// if the "ingest_in_progress" file exists, return processing
	if _, err := os.Stat(ingestDir("ingest_in_progress")); err == nil {
		w.WriteHeader(http.StatusProcessing)
		w.Write([]byte("Ingest in progress"))
		return
	}
	// otherwise return 200
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ingest complete"))
}

func CheckIngestFlag() bool {
	_, err := os.Stat(ingestDir("ingest_in_progress"))
	return err == nil
}

func StartIngestFlag() {
	_ = os.WriteFile(ingestDir("ingest_in_progress"), []byte{}, 0644)
}

func EndIngestFlag() {
	err := os.Remove(ingestDir("ingest_in_progress"))
	if err != nil {
		slog.Error("Error removing ingest_in_progress: %v", err)
	}
}

func IngestManifest() error {
	// Create the processing flag file
	defer os.Remove("ingest_in_progress")

	slog.Info("Reading manifest.json")
	b, err := os.ReadFile(ingestDir("manifest.json"))
	if err != nil {
		panic(err)
	}
	var manifest activedirectory.AssociationManifest
	err = json.Unmarshal(b, &manifest)
	if err != nil {
		panic(err)
	}
	slog.Info("Manifest read successfully")

	slog.Info("Creating coldfront objects")
	var users []User
	for _, u := range manifest.Users {
		// if slices.Contains(filter_usernames, u.Username) {
		// 	continue
		// }
		users = append(users, NewUser(u.Username, u.Firstname, u.Lastname))
	}
	var projects []Project
	for _, p := range manifest.Projects {
		projects = append(projects, NewProject(p.Name, p.Owner))
	}
	slog.Info("Coldfront objects created successfully")

	slog.Info("Creating associations")
	var associations []Association
	for _, p := range manifest.Projects {
		for _, u := range p.Users {
			if p.Owner == u {
				continue
			}
			a := NewAssociation(u, p.Name, p.Owner)
			if slices.Contains(p.Admins, u) {
				a.SetManager()
			}
			associations = append(associations, a)
		}
	}
	slog.Info("Associations created successfully")

	saveUsers(users)
	saveProjects(projects)
	saveAssociations(associations)

	cfLoadUsers()
	cfLoadProjects()
	res := cfLoadAssociations()
	if res == nil {
		return fmt.Errorf("error loading associations")
	}
	if res.Created != nil {
		for _, a := range res.Created {
			msg := fmt.Sprintf("Created association: %s -> %s", a.Username, a.Project)
			slog.Info(msg)
		}
	}
	if res.Removed != nil {
		for _, a := range res.Removed {
			msg := fmt.Sprintf("Removed association: %s -> %s", a.Username, a.Project)
			slog.Info(msg)
		}
	}
	return nil
}
