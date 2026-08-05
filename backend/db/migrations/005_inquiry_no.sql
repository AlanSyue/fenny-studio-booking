-- 1) 先加可為 NULL 的欄位，回填後再收緊約束
ALTER TABLE inquiries ADD COLUMN inquiry_no VARCHAR(20);

-- 2) 每日流水號計數表
CREATE TABLE inquiry_daily_counters (
    day      DATE PRIMARY KEY,
    last_seq INTEGER NOT NULL
);

-- 3) 回填既有詢問：依台北時區分日、同日內按建立時間排序
WITH numbered AS (
    SELECT id,
           (created_at AT TIME ZONE 'Asia/Taipei')::date AS tpe_day,
           ROW_NUMBER() OVER (
               PARTITION BY (created_at AT TIME ZONE 'Asia/Taipei')::date
               ORDER BY created_at, id
           ) AS seq
    FROM inquiries
)
UPDATE inquiries i
SET inquiry_no = 'FEN-' || to_char(n.tpe_day, 'YYYYMMDD') || '-' || lpad(n.seq::text, 3, '0')
FROM numbered n
WHERE i.id = n.id;

-- 4) 把各日已用掉的最大流水號寫進計數表，
--    避免回填後新產生的編號與回填編號碰撞
INSERT INTO inquiry_daily_counters (day, last_seq)
SELECT (created_at AT TIME ZONE 'Asia/Taipei')::date, count(*)
FROM inquiries
GROUP BY 1;

-- 5) 回填完成，收緊約束
ALTER TABLE inquiries ALTER COLUMN inquiry_no SET NOT NULL;
ALTER TABLE inquiries ADD CONSTRAINT inquiries_inquiry_no_key UNIQUE (inquiry_no);
