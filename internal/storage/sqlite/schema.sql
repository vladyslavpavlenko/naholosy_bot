-- The schema the first version of the bot created. Applied with IF NOT EXISTS
-- so that pointing the bot at an existing naholosy.db changes nothing.
CREATE TABLE IF NOT EXISTS naholosy (
    id             INTEGER PRIMARY KEY,
    letter         VARCHAR(1),
    word_lowercase VARCHAR(50),
    word           VARCHAR(25),
    hint           VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS users (
    user_id             INTEGER UNIQUE,
    menu_stage          VARCHAR(25),
    skip_tutorial       INTEGER,
    practice_mode       INTEGER,
    already_asked_words VARCHAR(700),
    asked_word          VARCHAR(25),
    correct_answers     INTEGER,
    wrong_answers       INTEGER,
    answered_count      INTEGER,
    combo_count         INTEGER
);

CREATE TABLE IF NOT EXISTS users_xp (
    user_id       INTEGER UNIQUE,
    learned_words VARCHAR(15000)
);
