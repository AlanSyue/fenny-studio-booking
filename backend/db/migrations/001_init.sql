CREATE TABLE admin_users (
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE courses (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    image_path  VARCHAR(500),
    description TEXT,
    is_active   BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE inquiries (
    id                 SERIAL PRIMARY KEY,
    name               VARCHAR(200),
    email              VARCHAR(255) NOT NULL,
    course_id          INTEGER REFERENCES courses(id),
    headcount          VARCHAR(50),
    additional_contact TEXT,
    is_processed       BOOLEAN DEFAULT FALSE,
    created_at         TIMESTAMPTZ DEFAULT NOW()
);
