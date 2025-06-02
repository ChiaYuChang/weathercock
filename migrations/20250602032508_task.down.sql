ALTER SEQUENCE users.tasks_id_seq RESTART WITH 1;

DROP TABLE IF EXISTS users.tasks;

DROP TYPE IF EXISTS source_type;

ALTER SEQUENCE users.info_id_seq RESTART WITH 1;

DROP TABLE IF EXISTS users.info;

DROP TYPE IF EXISTS task_status;