-- max_procs: max number of processes per user (prevents fork bombs)
-- io_read_bps/io_write_bps: I/O bandwidth limit in bytes/sec (0 = unlimited)
ALTER TABLE packages
    ADD COLUMN max_procs    INT    NOT NULL DEFAULT 200,
    ADD COLUMN io_read_bps  BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN io_write_bps BIGINT NOT NULL DEFAULT 0;
