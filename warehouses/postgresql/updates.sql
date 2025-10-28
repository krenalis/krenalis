ALTER TABLE "events" RENAME COLUMN "user" TO "muid";

ALTER TABLE "_user_identities" RENAME COLUMN "__gid__" TO "__muid__";

-- Note: replace "_users_0" with the name of the actual users table you have in
-- your data warehouse.
ALTER TABLE "_users_0" RENAME COLUMN "__id__" to "__muid__";

ALTER VIEW "users" RENAME COLUMN "__id__" to "__muid__";
