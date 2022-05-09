package reports

import (
	"bytes"
	"hash/fnv"
	"io"
	"log"
	"text/template"
	"time"

	"github.com/jackc/pgx"
	"github.com/karashiiro/operator/pkg/html"
	"github.com/karashiiro/operator/pkg/outlook"
	"github.com/karashiiro/operator/pkg/repos/plogons"
)

type ReportJob struct {
	Pool *pgx.ConnPool
}

func (j *ReportJob) Execute() {
	// Process all open pull requests
	reportTemplates, err := GetPlogonReportTemplates()
	if err != nil {
		log.Printf("Failed to retrieve plogons: %v\n", err)
		return
	}

	// Retrieve all of the active report readers
	readerConn, err := j.Pool.Acquire()
	if err != nil {
		log.Printf("Failed to acquire database connection: %v\n", err)
		return
	}
	defer j.Pool.Release(readerConn)

	rows, err := getReadersToNotify(readerConn)
	if err != nil {
		log.Printf("Unable to retrieve readers: %v\n", err)
		return
	}
	defer rows.Close()

	// Open a connection to handle inserting records into the report table
	// while we're iterating the results of the reader query
	reportConn, err := j.Pool.Acquire()
	if err != nil {
		log.Printf("Failed to acquire database connection: %v\n", err)
		return
	}
	defer j.Pool.Release(reportConn)

	for rows.Next() {
		// Read the next row from the database
		var readerId int
		var readerEmail string
		var readerGithub *string
		var readerLastSent *time.Time
		err := rows.Scan(&readerId, &readerEmail, &readerGithub, &readerLastSent)
		if err != nil {
			log.Printf("Unable to scan reader row: %v\n", err)
			continue
		}

		// Filter the updates since this reader's last email
		ref := time.Time{}
		if readerLastSent != nil {
			ref = *readerLastSent
		}

		// Figure out the size of the array we need so we can allocate
		// it all at once
		sinceLastEmail := 0
		for _, rt := range reportTemplates {
			if ref.IsZero() || rt.Plogon.Updated.After(ref) {
				sinceLastEmail++
			}
		}

		// Filter the stuff
		plogonsFiltered := make([]*ReportTemplate, sinceLastEmail)
		plogonsFilteredIdx := 0
		for _, rt := range reportTemplates {
			if ref.IsZero() || rt.Plogon.Updated.After(ref) {
				plogonsFiltered[plogonsFilteredIdx] = rt
				plogonsFilteredIdx++
			}
		}

		// If the result has no data, don't send an email for this interval
		if plogonsFilteredIdx == 0 {
			log.Println("Reader has no updates, skipping this interval")

			_, err := storeReportLogSkipped(reportConn, readerId)
			if err != nil {
				log.Printf("Unable to store report log: %v\n", err)
			}

			continue
		}

		// Send the email
		var readerMessage bytes.Buffer
		err = buildTemplate(&readerMessage, plogonsFiltered)
		if err != nil {
			log.Printf("Failed to build template: %v\n", err)
			continue
		}

		log.Printf("Sending email to %s\n", readerEmail)
		err = outlook.SendEmail(readerEmail, "Updated Dalamud Plugin Pull Requests", readerMessage.String())
		if err != nil {
			log.Printf("Unable to send mail: %v\n", err)
			continue
		}

		_, err = storeReportLog(reportConn, readerId)
		if err != nil {
			log.Printf("Unable to store report log: %v\n", err)
			continue
		}
	}

	if rows.Err() != nil {
		log.Printf("Error occurred while processing readers: %v\n", rows.Err())
		return
	}
}

func (j *ReportJob) Description() string {
	return "ReportJob"
}

func (j *ReportJob) Key() int {
	h := fnv.New32a()
	h.Write([]byte(j.Description()))
	return int(h.Sum32())
}

func GetPlogonReportTemplates() ([]*ReportTemplate, error) {
	// Retrieve all open pull requests
	plogonList, plogonPRs, err := plogons.GetPlogons()
	if err != nil {
		return nil, err
	}

	// Get the validation states of all open pull requests
	plogonValidation := make([]*ReportPlogonValidationState, len(plogonList))
	for i, pr := range plogonPRs {
		log.Printf("Validating pull request #%d\n", pr.GetNumber())
		res, err := plogons.ValidatePullRequest(pr)
		plogonValidation[i] = &ReportPlogonValidationState{
			Result: res,
			Err:    err,
		}
	}

	// Zip the two slices so we can enumerate them together
	plogonTemplates := make([]*ReportTemplate, len(plogonList))
	for i, p := range plogonList {
		plogonTemplates[i] = &ReportTemplate{
			Plogon:          p,
			ValidationState: plogonValidation[i],
		}
	}

	return plogonTemplates, nil
}

func buildTemplate(w io.Writer, reportTemplates []*ReportTemplate) error {
	// Build the HTML template
	t, err := template.New("report.gohtml").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format(time.RFC822)
		},
	}).ParseFS(html.Files, "report.gohtml")
	if err != nil {
		return err
	}

	// Execute the template
	err = t.Execute(w, struct {
		PlogonStates []*ReportTemplate
	}{
		PlogonStates: reportTemplates,
	})
	if err != nil {
		return err
	}

	return nil
}

func getReadersToNotify(conn *pgx.Conn) (*pgx.Rows, error) {
	return conn.Query(`
		SELECT Reader.id, Reader.email, Reader.github, max(Report.sent_time)
		FROM Reader
		LEFT JOIN Report
			ON Reader.id = Report.reader_id
		WHERE active
		GROUP BY Reader.id
		HAVING count(Report.reader_id) = 0
			OR max(Report.sent_time) + Reader.report_interval <= now();
	`)
}

func storeReportLog(conn *pgx.Conn, readerId int) (int64, error) {
	tag, err := conn.Exec(`
		INSERT INTO Report (sent_time, reader_id, skipped)
		VALUES
			(now(), $1, FALSE);
	`, readerId)
	if err != nil {
		return 0, err
	}

	return tag.RowsAffected(), nil
}

func storeReportLogSkipped(conn *pgx.Conn, readerId int) (int64, error) {
	tag, err := conn.Exec(`
		INSERT INTO Report (sent_time, reader_id, skipped)
		VALUES
			(now(), $1, TRUE);
	`, readerId)
	if err != nil {
		return 0, err
	}

	return tag.RowsAffected(), nil
}
