// Package responses holds the text the bot sends. Everything is parsed as
// HTML, so anything interpolated from user input must be escaped.
package responses

// Buttons on the reply keyboards. They double as the commands the router
// matches on, since the bot is driven by reply keyboards rather than by
// callbacks.
const (
	FasterButton     = "😼 Хутчіш!"
	PracticeButton   = "🎯 Практика"
	AllWordsButton   = "🗂 Усі слова"
	DownloadButton   = "📎 Завантажити PDF"
	YesButton        = "😼 Так!"
	MenuButton       = "↩️ Меню"
	BackButton       = "↩️ Назад"
	FinishGameButton = "🏳️ Завершити тренування"
)

const Start = `Привіт, друже!

Я допоможу тобі вивчити наголоси, які треба знати для того, щоб скласти ЗНО з української мови на всі 200! 😸`

const SomethingWentWrong = `<b>😿 Ой-йой!</b>

Здається, щось пішло не так. Спробуй ще раз.`

const MainMenu = `<b>🏠 Головне меню</b>

🆒 Напиши мені слово, а я вкажу який у ньому наголос (якщо воно є у затвердженому переліку).
<b>Ти: </b><i>фольга</i>
<b>Я: </b><i>фОльга</i>

🔠 Напиши мені букву або кілька букв через пробіл, щоб побачити усі наголоси на цю літеру.
<b>Ти: </b><i>Є Я</i>
<b>Я: </b><i>єретИк
ярмаркОвий</i>`

const WordsMenu = `<b>🗂 Усі слова</b>

Обери букви на клавіатурі нижче, щоб побачити слова зі списку.`

const PracticeMenu = `<b>🎯 Практика</b>

Обери кількість слів для самоперевірки.`

const PracticeExplanation1 = "<b>%s? Чудовий вибір!</b>\n\n" +
	"Гаразд, зараз я надсилатиму тобі по одному слову, написаному маленькими літерами, " +
	"а ти обиратимеш правильний наголос (він позначається великою літерою) на клавіатурі."

const PracticeExplanation2 = `Не квапся і гарно подумай. Я не обмежуватиму тебе у часі.`

const PracticeExplanation3 = `Поїхали?`

// QuizProgressPattern takes the question number and the run length. It goes in
// the poll's description, under the question, so that the question itself is
// nothing but the word.
const QuizProgressPattern = `%d / %d`

// RunStarted introduces a run and puts up the standing keyboard.
const RunStarted = `<b>🎯 Розпочинаємо!</b>

Обирай наголос під словом.`

// MissedWords takes the words the run got wrong.
const MissedWords = `<b>📌 Варто повторити:</b>
%s`

// ResultsPattern takes the heading, then the number of answers, the run
// length, and the correct and wrong tallies.
const ResultsPattern = `%s

Всього відповідей <b>%d / %d</b>

✅ Правильно – <b>%d</b>
❌ Неправильно – <b>%d</b>`

// Headings a finished run can carry. A run the user walked away from says so
// in place of the usual one, rather than in a message of its own.
const (
	ResultsFinished = `<b>🏁 Тренування завершено!</b>`
	ResultsTimedOut = `<b>⏳ Тренування зупинено!</b>

Схоже, тебе немає поруч.`
)

// WordsHeader takes the letters that were asked for.
const WordsHeader = `<b>🗂 Усі слова</b> – %s`

const NoWordsForLetters = `У переліку немає слів на ці букви!`

const WordNotFound = `🤷 Такого слова немає у затвердженому переліку.`

const OutOfWords = `😿 Слова у переліку скінчилися. Завершую тренування.`
