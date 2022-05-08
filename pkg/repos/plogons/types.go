package plogons

import "time"

type PlogonLabel struct {
	Name  string
	Color string
}

type Plogon struct {
	Title     string
	URL       string
	Labels    []*PlogonLabel
	Submitter string
	Updated   time.Time
}

type PlogonMeta struct {
	Author                 string
	Name                   string
	Punchline              string
	Description            string
	Changelog              string
	Tags                   []string
	CategoryTags           []string
	IsHide                 bool
	InternalName           string
	AssemblyVersion        string
	TestingAssemblyVersion string
	IsTestingExclusive     bool
	RepoUrl                string
	ApplicableVersion      string
	ImageUrls              []string
	IconUrl                string
	DalamudApiLevel        int
	LoadPriority           int
}
