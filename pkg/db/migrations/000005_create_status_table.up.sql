BEGIN;

CREATE TABLE IF NOT EXISTS t_status(
		f_id INT,
		f_status TEXT PRIMARY KEY);

INSERT INTO t_status (f_id, f_status) VALUES (0, 'in queue to activation') ON CONFLICT DO NOTHING;
INSERT INTO t_status (f_id, f_status) VALUES (1, 'active') ON CONFLICT DO NOTHING;
INSERT INTO t_status (f_id, f_status) VALUES (2, 'exit') ON CONFLICT DO NOTHING;
INSERT INTO t_status (f_id, f_status) VALUES (3, 'slashed') ON CONFLICT DO NOTHING;

COMMIT;