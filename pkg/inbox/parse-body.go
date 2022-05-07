package inbox

import (
	"strings"
	"time"

	"github.com/jprobinson/eazye"
	"github.com/microcosm-cc/bluemonday"
)

type ReaderInfo struct {
	Email          string
	GitHub         string
	GitHubSet      bool
	ReportInterval time.Duration
}

func ParseBody(email eazye.Email, policy bluemonday.Policy) (*ReaderInfo, error) {
	r := &ReaderInfo{
		Email: policy.Sanitize(email.From.Address),
	}

	bodyLines := strings.Split(string(email.Text), "\n")
	for _, line := range bodyLines {
		lineCleaned := strings.TrimSpace(line)

		// Parse their optionally-provided GitHub username
		githubMatches := githubPattern.FindStringSubmatch(lineCleaned)
		if len(githubMatches) != 0 {
			github := githubMatches[githubPattern.SubexpIndex("github")]
			r.GitHub = policy.Sanitize(github)
			r.GitHubSet = true
			continue
		}

		// Parse their requested reporting interval
		intervalMatches := intervalPattern.FindStringSubmatch(lineCleaned)
		if len(intervalMatches) != 0 {
			interval, err := time.ParseDuration(intervalMatches[intervalPattern.SubexpIndex("interval")])
			if err == nil {
				r.ReportInterval = interval
				continue
			}
		}
	}

	return r, nil
}
