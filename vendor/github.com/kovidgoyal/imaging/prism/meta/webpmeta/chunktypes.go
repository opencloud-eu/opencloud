package webpmeta

var (
	chunkTypeRIFF = [4]byte{'R', 'I', 'F', 'F'}
	chunkTypeWEBP = [4]byte{'W', 'E', 'B', 'P'}
	chunkTypeVP8  = [4]byte{'V', 'P', '8', ' '}
	chunkTypeVP8L = [4]byte{'V', 'P', '8', 'L'}
	chunkTypeVP8X = [4]byte{'V', 'P', '8', 'X'}
	chunkTypeICCP = [4]byte{'I', 'C', 'C', 'P'}
)
