ALTER TABLE "events" RENAME COLUMN "user" TO "muid";

ALTER TABLE "_user_identities" RENAME COLUMN "__gid__" TO "__muid__";
