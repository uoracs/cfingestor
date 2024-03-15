package coldfront

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// {
//     "model": "project.projectuser",
//     "fields": {
//         "user": [
//             "cgray"
//         ],
//         "project": [
//             "Angular momentum in QGP holography",
//             "cgray"
//         ],
//         "role": [
//             "Manager"
//         ],
//         "status": [
//             "Active"
//         ],
//         "enable_notifications": true
//     }
// },

type Association struct {
	Fields AssociationFields `json:"fields"`
	Model  string            `json:"model"`
}

type AssociationFields struct {
	User                []string                `json:"user"`
	Project             AssociationProjectField `json:"project"`
	Role                []string                `json:"role"`
	Status              []string                `json:"status"`
	EnableNotifications bool                    `json:"enable_notifications"`
}

type AssociationProjectField struct {
	Name string
	PI   string
}

func (a AssociationProjectField) MarshalJSON() ([]byte, error) {
	l := []string{a.Name, a.PI}
	return json.Marshal(l)
}

func (a *AssociationProjectField) UnmarshalJSON(b []byte) error {
	var l []any
	err := json.Unmarshal(b, &l)
	if err != nil {
		return err
	}
	if len(l) != 2 {
		return fmt.Errorf("expected 2 elements, got %d", len(l))
	}
	a.Name = l[0].(string)
	a.PI = l[1].(string)
	return nil
}

func NewAssociation(user, project, owner string) Association {
	return Association{
		Fields: NewAssociationFields(user, project, owner),
		Model:  "project.projectuser",
	}
}

func (a Association) ToJSON() (string, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return "", fmt.Errorf("error marshalling association: %v", err)
	}
	return string(b), nil
}

func (a Association) FromJSON(s string) (Association, error) {
	var association Association
	err := json.Unmarshal([]byte(s), &association)
	if err != nil {
		return Association{}, fmt.Errorf("error unmarshalling association: %v", err)
	}
	return association, nil
}

type AssociationLoaderResponse struct {
	Created []struct {
		Project  string `json:"project"`
		Username string `json:"user"`
	} `json:"created"`
	Removed []struct {
		Project  string `json:"project"`
		Username string `json:"user"`
	} `json:"removed"`
}

func NewAssociationFields(user, project, owner string) AssociationFields {
	apf := AssociationProjectField{Name: project, PI: owner}
	return AssociationFields{
		User:                []string{user},
		Project:             apf,
		Role:                []string{"User"},
		Status:              []string{"Active"},
		EnableNotifications: false,
	}
}

func (a *Association) SetManager() {
	a.Fields.Role = []string{"Manager"}
}

func (a *Association) SetPI() {
	a.Fields.Role = []string{"PI"}
}

func (a Association) withNotifications(n bool) Association {
	a.Fields.EnableNotifications = n
	return a
}

func saveAssociations(associations []Association) {
	slog.Info("Saving associations")
	assocBytes, err := json.Marshal(associations)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(ingestDir("associations.json"), assocBytes, 0644)
	if err != nil {
		panic(err)
	}
}

func cfWriteLoaderScript() {
	slog.Info("Writing association loader script")
	script := CFImportAssociationsScript
	_ = os.WriteFile(ingestDir("cfloadassociations.py"), []byte(script), 0644)
}

// Loading associations is a little more complicated, there's a python wrapper that we need to use
// so we can use the django builtin stuff
func cfLoadAssociations() *AssociationLoaderResponse {
	cfWriteLoaderScript()
	slog.Info("Loading associations into coldfront")
	loadCommand := fmt.Sprintf("%s shell < %s", coldfrontCommand, ingestDir("cfloadassociations.py"))
	cmd := exec.Command("bash", "-c", loadCommand)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if cmd.ProcessState.ExitCode() != 0 {
		return nil
	}
    // this will show the output of the command
	if stderr.String() != "" {
		slog.Info(stderr.String())
	}
	var response AssociationLoaderResponse
	err := json.Unmarshal(stdout.Bytes(), &response)
	if err != nil {
		panic(err)
	}
	return &response
}
