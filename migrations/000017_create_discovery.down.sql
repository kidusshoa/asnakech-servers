DROP TRIGGER IF EXISTS courses_search_vector_trigger ON courses;
DROP FUNCTION IF EXISTS courses_search_vector_update();
DROP INDEX IF EXISTS courses_search_vector_idx;
ALTER TABLE courses DROP COLUMN IF EXISTS search_vector;

DROP TABLE IF EXISTS parent_student_links;
