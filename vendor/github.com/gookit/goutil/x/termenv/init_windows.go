package termenv

func init() {
	// terminal supports color OR noColor=true: Don't need to enable virtual process
	if colorLevel != TermColorNone || noColor {
		return
	}

	// try force enable colors on Windows terminal
	TryEnableVTP(needVTP)
}
