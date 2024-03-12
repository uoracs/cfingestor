package coldfront

import (
	"encoding/json"
	"fmt"
)

type User struct {
	Fields UserFields `json:"fields"`
	Model  string     `json:"model"`
}

func NewUser(username, firstname, lastname string) User {
	return User{
		Fields: NewUserFields(username, firstname, lastname),
		Model:  "auth.user",
	}
}

func (u User) ToJSON() (string, error) {
	b, err := json.Marshal(u)
	if err != nil {
		return "", fmt.Errorf("error marshalling user: %v", err)
	}
	return string(b), nil
}

func (u User) FromJSON(s string) (User, error) {
	var user User
	err := json.Unmarshal([]byte(s), &user)
	if err != nil {
		return User{}, fmt.Errorf("error unmarshalling user: %v", err)
	}
	return user, nil
}

type UserFields struct {
	Username    string `json:"username"`
	Firstname   string `json:"first_name"`
	Lastname    string `json:"last_name"`
	Email       string `json:"email"`
	IsActive    bool   `json:"is_active"`
	IsStaff     bool   `json:"is_staff"`
	IsSuperuser bool   `json:"is_superuser"`
}

func NewUserFields(username, firstname, lastname string) UserFields {
	domain := "uoregon.edu"
	return UserFields{
		Username:  username,
		Firstname: firstname,
		Lastname:  lastname,
		Email:     fmt.Sprintf("%s@%s", username, domain),
	}
}

func (u User) withEmail(e string) User {
	u.Fields.Email = e
	return u
}

func (u User) withActive(a bool) User {
	u.Fields.IsActive = a
	return u
}

func (u User) withStaff(s bool) User {
	u.Fields.IsStaff = s
	return u
}

func (u User) withSuperuser(s bool) User {
	u.Fields.IsSuperuser = s
	return u
}
