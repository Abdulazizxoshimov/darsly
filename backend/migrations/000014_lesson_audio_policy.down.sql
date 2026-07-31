ALTER TABLE lessons
    DROP COLUMN IF EXISTS mute_on_entry,
    DROP COLUMN IF EXISTS allow_self_unmute;
