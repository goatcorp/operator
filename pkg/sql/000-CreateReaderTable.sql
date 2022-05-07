CREATE TABLE IF NOT EXISTS Reader (
    id              SERIAL,
    github          VARCHAR(40) UNIQUE,
    email           VARCHAR(40) UNIQUE NOT NULL,
    report_interval INTERVAL    NOT NULL,
    active          BOOLEAN     NOT NULL,

    PRIMARY KEY (id)
);