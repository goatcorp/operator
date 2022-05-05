CREATE TABLE IF NOT EXISTS Report (
    id             SERIAL,
    sent_time      TIMESTAMP WITH TIME ZONE NOT NULL,
    reader_id      INTEGER                  NOT NULL,

    PRIMARY KEY (id),
    FOREIGN KEY (reader_id) REFERENCES Reader(id) DEFERRABLE INITIALLY DEFERRED
);