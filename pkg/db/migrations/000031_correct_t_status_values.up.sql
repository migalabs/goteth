-- Update t_status values to match other tables
INSERT INTO t_status VALUES(2, 'exited');
INSERT INTO t_status VALUES(3, 'slashed');
OPTIMIZE TABLE t_status FINAL;