package plogons

import (
	"context"

	"github.com/google/go-github/v44/github"
)

func GetPlogons() ([]*Plogon, []*github.PullRequest, error) {
	// Retrieve all open pull requests
	client := github.NewClient(nil)
	plogonPRs, _, err := client.PullRequests.List(context.Background(), "goatcorp", "DalamudPlugins", &github.PullRequestListOptions{
		State: "open",
	})
	if err != nil {
		return nil, nil, err
	}

	// Make the plogons :dognosepretty:
	plogonsPretty := make([]*Plogon, len(plogonPRs))
	for i, plogon := range plogonPRs {
		labels := make([]*PlogonLabel, len(plogon.Labels))
		for j, label := range plogon.Labels {
			labels[j] = &PlogonLabel{
				Name:  label.GetName(),
				Color: label.GetColor(),
			}
		}

		plogonsPretty[i] = &Plogon{
			Title:     plogon.GetTitle(),
			URL:       plogon.GetHTMLURL(),
			Labels:    labels,
			Submitter: plogon.User.GetLogin(),
			Updated:   plogon.GetUpdatedAt(),
		}
	}

	return plogonsPretty, plogonPRs, nil
}
