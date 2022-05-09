package reports

import "github.com/karashiiro/operator/pkg/repos/plogons"

type ReportPlogonValidationState struct {
	Result *plogons.PlogonMetaValidationResult
	Err    error
}

type ReportTemplate struct {
	Plogon          *plogons.Plogon
	ValidationState *ReportPlogonValidationState
}
