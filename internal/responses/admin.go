package responses

import (
	"fmt"
	"strings"
	"time"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/stats"
)

// Admin-facing texts. They are only ever sent to the IDs in ADMIN_IDS.
const (
	BroadcastUsage = `<b>Розсилка</b>

<code>/broadcast текст</code> — розіслати текст (HTML).
Або відповідь на повідомлення: <code>/broadcast</code> — розіслати його копію з усім форматуванням.

<code>/broadcast_test ...</code> — надіслати лише собі.
<code>/broadcast_confirm</code> — підтвердити, <code>/broadcast_cancel</code> — скасувати.`

	BroadcastTestSent = `☝️ Так виглядатиме розсилка.`
	BroadcastNothing  = `Немає підготовленої розсилки.`
	BroadcastCanceled = `Розсилку скасовано.`
)

// BroadcastPrepared takes the size of the audience.
const BroadcastPrepared = `☝️ Так виглядатиме розсилка.

Отримувачів: <b>%d</b>
<code>/broadcast_confirm</code> — надіслати, <code>/broadcast_cancel</code> — скасувати.`

// BroadcastReport takes the delivered, failed and total counts and the elapsed
// time.
const BroadcastReport = `<b>📣 Розсилку завершено</b>

Доставлено: <b>%d</b>
Не доставлено: <b>%d</b>
Усього: <b>%d</b> за %s`

// Status renders the admin report.
func Status(r stats.Report) string {
	var b strings.Builder

	b.WriteString("<b>📊 Статус</b>\n\n")

	b.WriteString("<b>Користувачі</b>\n")
	fmt.Fprintf(&b, "Усього: <b>%d</b>\n", r.Users)

	m := r.Metrics
	b.WriteString("\n<b>Цей запуск</b>\n")
	fmt.Fprintf(&b, "Аптайм: %s\n", duration(r.Runtime.Uptime))
	fmt.Fprintf(&b, "Рекавері панік: %d\n", m.PanicsRecovered)
	fmt.Fprintf(&b, "Оновлень: %d (помилок %d)\n", m.UpdatesTotal, m.UpdatesFailed)
	fmt.Fprintf(&b, "Пошук слів: %d знайдено, %d ні\n", m.LookupsHit, m.LookupsMiss)
	fmt.Fprintf(&b, "Розсилки: %d доставлено, %d ні\n", m.BroadcastsDelivered, m.BroadcastsFailed)

	b.WriteString("\n<b>Процес</b>\n")
	fmt.Fprintf(&b, "%s · %d горутин · %.1f МБ · %d слів\n",
		r.Runtime.GoVersion, r.Runtime.Goroutines, float64(r.Runtime.HeapAlloc)/(1<<20), r.CatalogSize)

	for _, h := range m.Handlers {
		fmt.Fprintf(&b, "· %s (avg. %s, min. %s, max. %s, помилок %d)\n",
			escape(h.Name), duration(h.Average), duration(h.Min), duration(h.Max), h.Errors)
	}

	return b.String()
}

// duration renders a duration without sub-millisecond noise.
func duration(d time.Duration) string {
	if d < time.Millisecond {
		return d.Round(time.Microsecond).String()
	}
	if d < time.Minute {
		return d.Round(time.Millisecond).String()
	}

	return d.Round(time.Second).String()
}
