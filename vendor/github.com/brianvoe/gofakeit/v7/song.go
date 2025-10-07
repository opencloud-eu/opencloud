package gofakeit

func SongName() string { return songName(GlobalFaker) }

func (f *Faker) SongName() string { return songName(f) }

func songName(f *Faker) string { return getRandValue(f, []string{"song", "name"}) }

func SongArtist() string { return songArtist(GlobalFaker) }

func (f *Faker) SongArtist() string { return songArtist(f) }

func songArtist(f *Faker) string { return getRandValue(f, []string{"song", "artist"}) }

func SongGenre() string { return songGenre(GlobalFaker) }

func (f *Faker) SongGenre() string { return songGenre(f) }

func songGenre(f *Faker) string { return getRandValue(f, []string{"song", "genre"}) }

type SongInfo struct {
	Name   string `json:"name" xml:"name"`
	Artist string `json:"artist" xml:"artist"`
	Genre  string `json:"genre" xml:"genre"`
}

func Song() *SongInfo { return song(GlobalFaker) }

func (f *Faker) Song() *SongInfo { return song(f) }

func song(f *Faker) *SongInfo {
	return &SongInfo{
		Name:   songName(f),
		Artist: songArtist(f),
		Genre:  songGenre(f),
	}
}

func addSongLookup() {
	AddFuncLookup("song", Info{
		Display:     "Song",
		Category:    "song",
		Description: "Song with a drum and horn instrumentation",
		Example: `{
	"name": "New Rules",
	"genre": "Tropical house"
}`,
		Output:      "map[string]string",
		ContentType: "application/json",
		Aliases: []string{
			"musical work",
			"sound piece",
			"single release",
			"record title",
			"audio selection",
		},
		Keywords: []string{
			"song", "music", "track", "tune", "melody",
			"artist", "genre", "name", "composition", "recording",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return song(f), nil
		},
	})

	AddFuncLookup("songname", Info{
		Display:     "Song Name",
		Category:    "song",
		Description: "Title or name of a specific song used for identification and reference",
		Example:     "New Rules",
		Output:      "string",
		Aliases: []string{
			"song title",
			"track name",
			"music title",
			"song label",
		},
		Keywords: []string{
			"song", "title", "name", "track", "music",
			"single", "hit", "tune", "recording",
			"composition", "melody",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return songName(f), nil
		},
	})

	AddFuncLookup("songartist", Info{
		Display:     "Song Artist",
		Category:    "song",
		Description: "The artist of maker of song",
		Example:     "Dua Lipa",
		Output:      "string",
		Aliases: []string{
			"performer name",
			"music act",
			"band name",
			"recording artist",
			"song creator",
		},
		Keywords: []string{
			"song", "artist", "singer", "musician", "composer",
			"band", "producer", "vocalist", "group", "instrumentalist",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return songArtist(f), nil
		},
	})

	AddFuncLookup("songgenre", Info{
		Display:     "Genre",
		Category:    "song",
		Description: "Category that classifies song based on common themes, styles, and storytelling approaches",
		Example:     "Action",
		Output:      "string",
		Aliases: []string{
			"music style",
			"song category",
			"musical classification",
			"sound type",
			"genre label",
		},
		Keywords: []string{
			"song", "genre", "style", "category", "type",
			"classification", "theme", "musical", "subgenre", "influence",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return songGenre(f), nil
		},
	})
}
