package coldfront

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

type Project struct {
	Fields ProjectFields `json:"fields"`
	Model  string        `json:"model"`
}

type ProjectFields struct {
	Description    string   `json:"description"`
	FieldOfScience []string `json:"field_of_science"`
	ForceReview    bool     `json:"force_review"`
	Pi             []string `json:"pi"`
	RequiresReview bool     `json:"requires_review"`
	Status         []string `json:"status"`
	Title          string   `json:"title"`
}

func NewProject(title, pi string) Project {
	return Project{
		Fields: NewProjectFields(title, pi),
		Model:  "project.project",
	}
}

func (p Project) ToJSON() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("error marshalling project: %v", err)
	}
	return string(b), nil
}

func (p Project) FromJSON(s string) (Project, error) {
	var project Project
	err := json.Unmarshal([]byte(s), &project)
	if err != nil {
		return Project{}, fmt.Errorf("error unmarshalling project: %v", err)
	}
	return project, nil
}

func NewProjectFields(title, pi string) ProjectFields {
	return ProjectFields{
		Description:    "",
		Title:          title,
		ForceReview:    false,
		RequiresReview: true,
		FieldOfScience: []string{"Other"},
		Pi:             []string{pi},
		Status:         []string{"Active"},
	}
}

func (p Project) withDescription(d string) Project {
	p.Fields.Description = d
	return p
}

func (p Project) withFieldOfScience(f []string) Project {
	p.Fields.FieldOfScience = f
	return p
}

func (p Project) withForceReview(f bool) Project {
	p.Fields.ForceReview = f
	return p
}

func (p Project) withPi(pi []string) Project {
	p.Fields.Pi = pi
	return p
}

func (p Project) withRequiresReview(r bool) Project {
	p.Fields.RequiresReview = r
	return p
}

func (p Project) withStatus(s []string) Project {
	p.Fields.Status = s
	return p
}

func saveProjects(projects []Project) {
	slog.Info("Saving projects")
	projectBytes, err := json.Marshal(projects)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(ingestDir("projects.json"), projectBytes, 0644)
	if err != nil {
		panic(err)
	}
}

// Loading projects is pretty easy, we don't need to do anything special
func cfLoadProjects() {
	slog.Info("Loading projects into coldfront")
	cmd := exec.Command(coldfrontCommand, "loaddata", "--format=json", ingestDir("projects.json"))
	_ = cmd.Run()
}
