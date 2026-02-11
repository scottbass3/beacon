package registry

import (
	"net/url"
)

type RegistryV2Provider struct{}

func (RegistryV2Provider) Kind() string {
	return "registry_v2"
}

func (RegistryV2Provider) TableSpec() TableSpec {
	return TableSpec{
		SupportsProjects: false,
		Image: ImageTableSpec{
			ShowTagCount: false,
			ShowPulls:    false,
			ShowUpdated:  false,
		},
		Tag: TagTableSpec{
			ShowSize:       false,
			ShowPushed:     false,
			ShowLastPulled: false,
		},
		History: HistoryTableSpec{
			ShowSize:    true,
			ShowComment: true,
		},
	}
}

func (RegistryV2Provider) NeedsAuthPrompt(auth Auth) bool {
	if auth.Kind == "none" {
		return false
	}
	if auth.RegistryV2.Anonymous {
		return false
	}
	if auth.RegistryV2.Username == "" {
		return true
	}
	if auth.RegistryV2.Password == "" && !(auth.RegistryV2.Remember && auth.RegistryV2.RefreshToken != "") {
		return true
	}
	return false
}

func (RegistryV2Provider) AuthUI(auth Auth) AuthUI {
	if auth.Kind == "none" || auth.RegistryV2.Anonymous {
		return AuthUI{}
	}
	return AuthUI{
		ShowUsername: true,
		ShowPassword: true,
		ShowRemember: true,
	}
}

func (RegistryV2Provider) PrepareAuth(baseURL *url.URL, auth *Auth) error {
	if auth.Kind == "" || auth.Kind == "none" {
		auth.Kind = "registry_v2"
		auth.RegistryV2.Anonymous = true
	}
	if auth.RegistryV2.Service == "" && baseURL != nil && baseURL.Host != "" {
		auth.RegistryV2.Service = baseURL.Host
	}
	return nil
}

func (RegistryV2Provider) NewClient(baseURL *url.URL, auth Auth, logger RequestLogger) (Client, error) {
	return newRegistryV2Client(baseURL, auth, logger), nil
}
