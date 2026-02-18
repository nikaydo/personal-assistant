package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type User struct {
	Self         string     `json:"self"`
	AccountID    string     `json:"accountId"`
	AccountType  string     `json:"accountType"`
	EmailAddress string     `json:"emailAddress"`
	AvatarURLs   AvatarURLs `json:"avatarUrls"`
	DisplayName  string     `json:"displayName"`
	Active       bool       `json:"active"`
	TimeZone     string     `json:"timeZone"`
	Locale       string     `json:"locale"`
}

type AvatarURLs struct {
	Size48 string `json:"48x48"`
	Size24 string `json:"24x24"`
	Size16 string `json:"16x16"`
	Size32 string `json:"32x32"`
}

func (j *Jira) GetUserInfo(url string) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/3/user/search?query=%s", url, j.Email), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(j.Email+":"+j.ApiToken)))
	req.Header.Add("Accept", "application/json")
	resp, err := j.Client.Do(req)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var users []User
	err = json.Unmarshal(body, &users)
	if err != nil {
		return err
	}
	j.User = users
	j.MainUser = users[0]
	return nil
}
