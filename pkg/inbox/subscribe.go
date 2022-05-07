package inbox

import (
	"bytes"
	"io"
	"log"
	"text/template"
	"time"

	"github.com/jackc/pgx"
	"github.com/karashiiro/operator/pkg/html"
	"github.com/karashiiro/operator/pkg/outlook"
)

func saveSubscribers(conn *pgx.Conn, readers []*ReaderInfo) {
	for _, r := range readers {
		_, err := storeReader(conn, r)
		if err != nil {
			log.Printf("Failed to add new reader: %v\n", err)
			continue
		}

		log.Printf("Sending subscription confirmation email to %s\n", r.Email)

		var subscribeMessage bytes.Buffer
		err = buildSubscribeTemplate(&subscribeMessage, r.ReportInterval)
		if err != nil {
			log.Printf("Failed to build subscribe template: %v\n", err)
		}

		err = outlook.SendEmail(r.Email, "Subscription confirmed", subscribeMessage.String())
		if err != nil {
			log.Printf("Unable to send mail: %v\n", err)
			continue
		}

		log.Printf("Added reader %s\n", r.Email)
	}
}

func buildSubscribeTemplate(w io.Writer, interval time.Duration) error {
	t, err := template.ParseFS(html.Files, "confirm-subscribe.gohtml")
	if err != nil {
		return err
	}

	err = t.Execute(w, struct {
		Interval time.Duration
	}{
		Interval: interval,
	})
	if err != nil {
		return err
	}

	return nil
}

func storeReader(conn *pgx.Conn, r *ReaderInfo) (int64, error) {
	t, err := conn.Exec(`
		INSERT INTO Reader (email, github, report_interval, active)
		VALUES
			($1, $2, $3, TRUE)
	`, r.Email, r.GitHub, r.ReportInterval)
	if err != nil {
		return 0, err
	}

	return t.RowsAffected(), nil
}
