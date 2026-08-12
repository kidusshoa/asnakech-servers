DROP TABLE IF EXISTS enrollment_events;
DROP TABLE IF EXISTS enrollments;
DROP TABLE IF EXISTS enrollment_invite_codes;

ALTER TABLE courses
    DROP CONSTRAINT IF EXISTS courses_enrollment_capacity_pos;

ALTER TABLE courses
    DROP COLUMN IF EXISTS waitlist_enabled,
    DROP COLUMN IF EXISTS enrollment_open,
    DROP COLUMN IF EXISTS enrollment_capacity;
