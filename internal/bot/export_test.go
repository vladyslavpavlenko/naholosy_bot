package bot

// Handles reports whether a route has a handler behind it. A route without one
// would be a nil call, which the test below guards against.
func (b *Bot) Handles(route string) bool {
	_, ok := b.routes[route]

	return ok
}

// Routes returns every route the router can resolve to.
func Routes() []string {
	return []string{
		RouteStart, RouteMenu, RouteWordsMenu, RouteLetters, RouteLookup, RouteDownload,
		RoutePracticeMenu, RouteChooseSize, RouteStartRun, RouteAnswer, RouteFinishRun,
		RouteStatus, RouteBroadcast, RouteBroadcastTest, RouteBroadcastConfirm, RouteBroadcastCancel,
	}
}
