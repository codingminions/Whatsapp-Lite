DROP INDEX IF EXISTS idx_group_members_group_id;
DROP INDEX IF EXISTS idx_group_members_user_id;
DROP INDEX IF EXISTS idx_group_members_role;
DROP TABLE IF EXISTS group_members;
DROP TYPE IF EXISTS member_role;