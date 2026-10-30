package data

// Hacker consists of random hacker phrases
var Hacker = map[string][]string{
	"abbreviation": {"TCP", "HTTP", "SDD", "RAM", "GB", "CSS", "SSL", "AGP", "SQL", "FTP", "PCI", "AI", "ADP", "RSS", "XML", "EXE", "COM", "HDD", "THX", "SMTP", "SMS", "USB", "PNG", "SAS", "IB", "SCSI", "JSON", "XSS", "JBOD"},
	"adjective":    {"auxiliary", "primary", "back-end", "digital", "open-source", "virtual", "cross-platform", "redundant", "online", "haptic", "multi-byte", "bluetooth", "wireless", "1080p", "neural", "optical", "solid state", "mobile"},
	"noun":         {"driver", "protocol", "bandwidth", "panel", "microchip", "program", "port", "card", "array", "interface", "system", "sensor", "firewall", "hard drive", "pixel", "alarm", "feed", "monitor", "application", "transmitter", "bus", "circuit", "capacitor", "matrix"},
	"verb":         {"back up", "bypass", "hack", "override", "compress", "copy", "navigate", "index", "connect", "generate", "quantify", "calculate", "synthesize", "input", "transmit", "program", "reboot", "parse", "read", "write", "load", "render", "validate", "verify", "sign", "decrypt", "encrypt", "construct", "deconstruct", "compile", "transpile", "bundle", "lock", "unlock", "buffer", "format"},
	"ingverb":      {"backing up", "bypassing", "hacking", "overriding", "compressing", "copying", "navigating", "indexing", "connecting", "generating", "quantifying", "calculating", "synthesizing", "transmitting", "programming", "parsing"},
	"phrase": {
		"If we {hackerverb} the {hackernoun}, we can get to the {hackerabbreviation} {hackernoun} through the {hackeradjective} {hackerabbreviation} {hackernoun}!",
		"We need to {hackerverb} the {hackeradjective} {hackerabbreviation} {hackernoun}!",
		"Try to {hackerverb} the {hackerabbreviation} {hackernoun}, maybe it will {hackerverb} the {hackeradjective} {hackernoun}!",
		"You can't {hackerverb} the {hackernoun} without {hackeringverb} the {hackeradjective} {hackerabbreviation} {hackernoun}!",
		"Use the {hackeradjective} {hackerabbreviation} {hackernoun}, then you can {hackerverb} the {hackeradjective} {hackernoun}!",
		"The {hackerabbreviation} {hackernoun} is down, {hackerverb} the {hackeradjective} {hackernoun} so we can {hackerverb} the {hackerabbreviation} {hackernoun}!",
		"{hackeringverb} the {hackernoun} won't do anything, we need to {hackerverb} the {hackeradjective} {hackerabbreviation} {hackernoun}!",
		"I'll {hackerverb} the {hackeradjective} {hackerabbreviation} {hackernoun}, that should {hackerverb} the {hackerabbreviation} {hackernoun}!",
	},
}
