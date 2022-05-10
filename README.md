# Caprine Operator
*The* goatcorp email reporting service.
![](assets/email.png)

## Environment variables
* `OPERATOR_EMAIL`: The email address to use for sending emails.
* `OPERATOR_PASSWORD`: The password corresponding to the email address.
* `OPERATOR_SMTP_SERVER`: The SMTP server to be used for sending emails.
* `OPERATOR_IMAP_SERVER`: The IMAP server to be used for receiving emails.
* `OPERATOR_POSTGRES`: The PostgreSQL host server override (optional). Defaults to `localhost`. If the application is being run inside of a Docker container, this needs to be overriden.
* `OPERATOR_INBOX`: The inbox that should be used for emails sent to Caprine Operator.
* `OPERATOR_JUNK`: The junk email folder for the Operator's email account. Note that Outlook names this folder `Junk` internally, despite showing `Junk Email` as the folder name to users.

The SMTP and IMAP servers for Outlook can be found [here](https://support.microsoft.com/en-us/office/pop-imap-and-smtp-settings-for-outlook-com-d088b986-291d-42b8-9564-9c414e2aa040).

## Notes for admins
The Operator checks *unread* emails periodically for user interactions. Please refrain from checking the Operator's unread emails manually (read emails are fine).
