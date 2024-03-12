package coldfront

import (
	"encoding/json"
	"fmt"
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
	User                []string `json:"user"`
	Project             []string `json:"project"`
	Role                []string `json:"role"`
	Status              []string `json:"status"`
	EnableNotifications bool     `json:"enable_notifications"`
}

func NewAssociation(user, project string) Association {
	return Association{
		Fields: NewAssociationFields(user, project),
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

func NewAssociationFields(user, project string) AssociationFields {
	return AssociationFields{
		User:                []string{user},
		Project:             []string{project, user},
		Role:                []string{""},
		Status:              []string{"Active"},
		EnableNotifications: false,
	}
}

func (a Association) withManager() Association {
	a.Fields.Role = []string{"Manager"}
	return a
}

func (a Association) withNotifications(n bool) Association {
	a.Fields.EnableNotifications = n
	return a
}
