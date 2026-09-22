# naholosy_bot

A Telegram bot that helps memorise the Ukrainian stress patterns required by
the national exam. It answers "where is the stress in this word?", lists the
approved words letter by letter, and runs timed-free practice sessions of 12,
24 or 36 words.

This is a rewrite of the original bot. It speaks to the same `naholosy.db`
file with the same schema, so replacing the binary is the whole upgrade — no
migration, no data loss.

## Running it

```bash
cp .env.example .env   # fill in BOT_TOKEN and ADMIN_IDS
task run
```

| Variable    | Meaning                                                         |
|-------------|-----------------------------------------------------------------|
| `BOT_TOKEN` | Bot token from [@BotFather](https://t.me/BotFather).             |
| `LOG_LEVEL` | `DEBUG` for human-readable logs, `PROD` for production levels.   |
| `ADMIN_IDS` | Comma-separated Telegram user IDs allowed to run admin commands. |
| `DB_PATH`   | SQLite file. Defaults to `naholosy.db` in the working directory. |

An empty or missing database is created and seeded with the word list on
first start; an existing one is left exactly as it is.

```bash
task build      # static binary in ./bin, CGO_ENABLED=0
task test:unit  # tests
task test:race  # tests under the race detector
task lint       # golangci-lint
```

## Layout

Each domain owns its own package: the entities, the rules that act on them and
the ports they need. Nothing in a domain package knows about Telegram or
SQLite.

```
main/                    entry point: config, signals
internal/
  app/                   composition root
  bot/                   Telegram runtime: routing, per-user locking, recovery
  handlers/              one update in, messages out
  keyboard/              reply keyboards
  responses/             every string the bot sends
  sender/                outgoing messages, quiz polls, retries
  accent/                the word list and the index over it
  user/                  who is talking and where they are in the conversation
  practice/              the training run and its rules
  broadcast/             sending one message to everybody
  stats/                 the /status report
  words/                 stress parsing, variants, text predicates
  storage/sqlite/        the repositories, over the original schema
pkg/logger/              zap wrapper
```

The conversation is a state machine. The same button means different things
depending on where the user is, so `bot.Resolve` maps (stage, text) to a
handler in one place, and is a pure function so the whole routing table is
covered by tests.

A practice question is a native quiz poll, which lets Telegram mark the right
and wrong option on the client the moment the user picks. A poll answer comes
back with nothing but the poll's ID and the index chosen, so the options are
held in memory until the question is answered; after a restart an open question
can no longer be graded and the run moves on to the next word.

## Admin commands

Available to the IDs in `ADMIN_IDS`, anywhere in the conversation.

| Command                    | Effect                                                     |
|----------------------------|------------------------------------------------------------|
| `/status`                  | Users, learning progress, and this process's counters.     |
| `/broadcast <text>`        | Stage a broadcast written in Telegram HTML.                |
| `/broadcast` (as a reply)  | Stage a copy of the message replied to, formatting intact. |
| `/broadcast_test ...`      | Send it to yourself only.                                  |
| `/broadcast_confirm`       | Send the staged broadcast to everyone.                     |
| `/broadcast_cancel`        | Discard it.                                                |

Nothing leaves the chat until `/broadcast_confirm`. A staged broadcast lives in
memory, so a restart discards it rather than sending it by surprise.

## Notes on the rewrite

The behaviour users see is the original's, with the bugs taken out:

- The old bot panicked on any unexpected row and took the process down with it.
  Every panic path is now an error, and a recovery middleware keeps one bad
  update from ending the process.
- Drawing a practice word retried itself recursively until it found an unasked
  one. It now builds the eligible set and draws once.
- Telegram delivers a user's updates concurrently and the old bot handled each
  in a bare goroutine over shared mutable state. Updates are now serialised per
  user.
- Looking up an unknown word produced an empty message, which Telegram rejects.
  It now says the word is not on the list.
- Word lists were built by string-concatenating SQL. The word list is static,
  so it is read once and indexed in memory, which also avoids SQLite's
  ASCII-only case folding.
- The stickers, and the combo streak whose only purpose was to unlock them,
  are gone. Telegram marks the answer itself, and the run still ends with an
  emoji. The `combo_count` column stays in the file, untouched and unread.
- The old bot slept between messages. Nothing blocks now; messages are simply
  sent in order.
- A word answered wrongly goes back in the pool, so the run may put it again.
  Getting something wrong and never seeing it again is the worst thing a drill
  can do. The results name what was missed.
- Nothing in the bot panics. Configuration errors are reported and exit, update
  handling is wrapped in recovery, and the one goroutine the bot starts on its
  own recovers separately, since the handler's recovery does not reach it.
