    -- Revert t_status values to original state
    INSERT INTO t_status VALUES(2, 'slashed');
    INSERT INTO t_status VALUES(3, 'exited');
    OPTIMIZE TABLE t_status FINAL;