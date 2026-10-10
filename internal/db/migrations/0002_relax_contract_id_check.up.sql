ALTER TABLE contracts DROP CONSTRAINT IF EXISTS contracts_id_format;
ALTER TABLE contracts ADD CONSTRAINT contracts_id_format
    CHECK (id ~ '^[0-9]{4}(-[A-Za-zА-Яа-я0-9]{1,5})?$');