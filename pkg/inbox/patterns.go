package inbox

import "regexp"

var githubPattern = regexp.MustCompile(`(?i)github:\s*(?P<github>\S*)`)
var intervalPattern = regexp.MustCompile(`(?i)interval:\s*(?P<interval>\S*)`)
