package registry

import "net/url"

type HarborProvider struct{}

func (HarborProvider) Kind() string {
	return "harbor"
}

func (HarborProvider) TableSpec() TableSpec {
	return TableSpec{
		SupportsProjects: true,
		Image: ImageTableSpec{
			ShowTagCount: true,
			ShowPulls:    true,
			ShowUpdated:  true,
		},
		Tag: TagTableSpec{
			ShowSize:       true,
			ShowPushed:     true,
			ShowLastPulled: true,
		},
		History: HistoryTableSpec{
			ShowSize:    true,
			ShowComment: true,
		},
	}
}

func (HarborProvider) NeedsAuthPrompt(auth Auth) bool {
	if auth.Kind == "none" {
		return false
	}
	if auth.Harbor.Anonymous {
		return false
	}
	return auth.Harbor.Username == "" || auth.Harbor.Password == ""
}

func (HarborProvider) AuthUI(auth Auth) AuthUI {
	if auth.Kind == "none" || auth.Harbor.Anonymous {
		return AuthUI{}
	}
	return AuthUI{
		ShowUsername: true,
		ShowPassword: true,
		ShowRemember: false,
	}
}

func (HarborProvider) PrepareAuth(_ *url.URL, auth *Auth) error {
	if auth.Kind == "" {
		auth.Kind = "harbor"
	}
	return nil
}

func (HarborProvider) NewClient(baseURL *url.URL, auth Auth, logger RequestLogger) (Client, error) {
	return newHarborClient(baseURL, auth, logger), nil
}
