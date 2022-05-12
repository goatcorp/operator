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
	RepoURL                string `json:"RepoUrl"`
	ApplicableVersion      string
	ImageURLs              []string `json:"ImageUrls"`
	IconURL                string   `json:"IconUrl"`
	DalamudApiLevel        int
	LoadPriority           int
}

type PlogonMetaImageValidationResult struct {
	ImageExists bool
}

type PlogonMetaValidationResult struct {
	NameSet               bool
	InternalNameSet       bool
	DescriptionSet        bool
	AssemblyVersionSet    bool
	RepoURLSet            bool
	PunchlineSet          bool
	MatchesZipped         bool
	Testing               bool
	TestingHasTaggedTitle bool
	IconSet               bool
	IconExists            bool
	Images                []*PlogonMetaImageValidationResult
}
