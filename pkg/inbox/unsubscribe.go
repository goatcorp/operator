package inbox

import (
	"bytes"
	"io"
	"log"
	"text/template"

	"github.com/jackc/pgx"
	"github.com/karashiiro/operator/pkg/html"
	"github.com/karashiiro/operator/pkg/outlook"
)

func deleteUnsubscribers(conn *pgx.Conn, unsubscribers []string) {
	for _, us := range unsubscribers {
		_, err := deleteReader(conn, us)
		if err != nil {
			log.Printf("Failed to delete reader: %v\n", err)
			continue
		}

		var unsubscribeMessage bytes.Buffer
		err = buildUnsubscribeTemplate(&unsubscribeMessage)
		if err != nil {
			log.Printf("Failed to build unsubscribe template: %v\n", err)
			continue
		}

		err = outlook.SendEmail(us, "Unsubscribe confirmed", unsubscribeMessage.String())
		if err != nil {
			log.Printf("Unable to send mail: %v\n", err)
			continue
		}

		log.Printf("Deleted reader %s\n", us)
	}
}

func deleteReader(conn *pgx.Conn, addr string) (int64, error) {
	t1, err := conn.Exec("DELETE FROM Report WHERE reader_id = (SELECT id FROM Reader WHERE email = $1);", addr)
	if err != nil {
		return 0, err
	}

	t2, err := conn.Exec("DELETE FROM Reader WHERE email = $1;", addr)
	if err != nil {
		return t1.RowsAffected(), err
	}

	return t1.RowsAffected() + t2.RowsAffected(), nil
}

func buildUnsubscribeTemplate(w io.Writer) error {
	t, err := template.ParseFS(html.Files, "confirm-unsubscribe.gohtml")
	if err != nil {
		return err
	}

	err = t.Execute(w, struct{}{})
	if err != nil {
		return err
	}

	return nil
}
