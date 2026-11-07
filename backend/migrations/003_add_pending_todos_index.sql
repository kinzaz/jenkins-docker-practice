CREATE INDEX todos_pending_id_idx
    ON todos (id DESC)
    WHERE done = false;