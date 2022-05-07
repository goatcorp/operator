# Caprine Operator
Sends emails about things.
![](assets/email.png)

## Environment variables
* `OPERATOR_EMAIL`: The email address to use for sending emails.
* `OPERATOR_PASSWORD`: The password corresponding to the email address.
* `OPERATOR_SMTP_SERVER`: The SMTP server to be used for sending emails.
* `OPERATOR_IMAP_SERVER`: The IMAP server to be used for receiving emails.
* `OPERATOR_POSTGRES`: The PostgreSQL host server (optional). Defaults to `localhost`.
* `OPERATOR_INBOX`: The inbox that should be used for emails sent to Caprine Operator.
