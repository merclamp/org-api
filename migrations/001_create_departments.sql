-- +goose Up
CREATE TABLE departments (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(200) NOT NULL,
    parent_id  INTEGER REFERENCES departments(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_department_name_parent
        UNIQUE NULLS NOT DISTINCT (name, parent_id)
);

CREATE INDEX idx_departments_parent_id ON departments(parent_id);

-- +goose Down
DROP TABLE IF EXISTS departments;