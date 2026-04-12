-- 1) courses：加 product_url、woo_product_id (for upsert)
ALTER TABLE courses ADD COLUMN product_url VARCHAR(500);
ALTER TABLE courses ADD COLUMN woo_product_id INTEGER UNIQUE;

-- 2) inquiries：加地址、預算，改為多選 → course_id 保留但標示 legacy
ALTER TABLE inquiries ADD COLUMN city VARCHAR(50);
ALTER TABLE inquiries ADD COLUMN district VARCHAR(50);
ALTER TABLE inquiries ADD COLUMN address VARCHAR(255);
ALTER TABLE inquiries ADD COLUMN budget TEXT;

-- 3) 多對多 join table
CREATE TABLE inquiry_courses (
    inquiry_id INTEGER NOT NULL REFERENCES inquiries(id) ON DELETE CASCADE,
    course_id  INTEGER NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    PRIMARY KEY (inquiry_id, course_id)
);
CREATE INDEX idx_inquiry_courses_course ON inquiry_courses(course_id);
