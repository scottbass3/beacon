package registry

import "net/url"

type TableSpec struct {
	SupportsProjects bool
	Image            ImageTableSpec
	Tag              TagTableSpec
	History          HistoryTableSpec
}

type ImageTableSpec struct {
	ShowTagCount bool
	ShowPulls    bool
	ShowUpdated  bool
}

type TagTableSpec struct {
	ShowSize       bool
	ShowPushed     bool
	ShowLastPulled bool
}

type HistoryTableSpec struct {
	ShowSize    bool
	ShowComment bool
}

type AuthUI struct {
	ShowUsername bool
	ShowPassword bool
	ShowRemember bool
}

type Provider interface {
	Kind() string
	TableSpec() TableSpec
	NeedsAuthPrompt(auth Auth) bool
	AuthUI(auth Auth) AuthUI
	PrepareAuth(baseURL *url.URL, auth *Auth) error
	NewClient(baseURL *url.URL, auth Auth, logger RequestLogger) (Client, error)
}
