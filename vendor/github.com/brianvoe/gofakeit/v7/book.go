package gofakeit

func BookTitle() string { return bookTitle(GlobalFaker) }

func (f *Faker) BookTitle() string { return bookTitle(f) }

func bookTitle(f *Faker) string { return getRandValue(f, []string{"book", "title"}) }

func BookAuthor() string { return bookAuthor(GlobalFaker) }

func (f *Faker) BookAuthor() string { return bookAuthor(f) }

func bookAuthor(f *Faker) string { return getRandValue(f, []string{"book", "author"}) }

func BookGenre() string { return bookGenre(GlobalFaker) }

func (f *Faker) BookGenre() string { return bookGenre(f) }

func bookGenre(f *Faker) string { return getRandValue(f, []string{"book", "genre"}) }

type BookInfo struct {
	Title  string `json:"title" xml:"name"`
	Author string `json:"author" xml:"author"`
	Genre  string `json:"genre" xml:"genre"`
}

func Book() *BookInfo { return book(GlobalFaker) }

func (f *Faker) Book() *BookInfo { return book(f) }

func book(f *Faker) *BookInfo {
	return &BookInfo{
		Title:  bookTitle(f),
		Author: bookAuthor(f),
		Genre:  bookGenre(f),
	}
}

func addBookLookup() {
	AddFuncLookup("book", Info{
		Display:     "Book",
		Category:    "book",
		Description: "Written or printed work consisting of pages bound together, covering various subjects or stories",
		Example: `{
	"title": "Anna Karenina",
	"author": "Toni Morrison",
	"genre": "Thriller"
}`,
		Output:      "map[string]string",
		ContentType: "application/json",
		Aliases:     []string{"printed", "pages", "bound", "subjects", "stories", "literature", "text"},
		Keywords:    []string{"book", "written", "work", "consisting", "anna", "karenina", "toni", "morrison", "thriller"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return book(f), nil
		},
	})

	AddFuncLookup("booktitle", Info{
		Display:     "Title",
		Category:    "book",
		Description: "The specific name given to a book",
		Example:     "Hamlet",
		Output:      "string",
		Aliases:     []string{"title", "name", "specific", "given", "heading"},
		Keywords:    []string{"book", "identification", "hamlet", "naming", "designation", "label", "caption"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return bookTitle(f), nil
		},
	})

	AddFuncLookup("bookauthor", Info{
		Display:     "Author",
		Category:    "book",
		Description: "The individual who wrote or created the content of a book",
		Example:     "Mark Twain",
		Output:      "string",
		Aliases:     []string{"author", "writer", "creator", "individual", "content", "literary"},
		Keywords:    []string{"book", "wrote", "created", "mark", "twain", "composer", "originator", "penned"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return bookAuthor(f), nil
		},
	})

	AddFuncLookup("bookgenre", Info{
		Display:     "Genre",
		Category:    "book",
		Description: "Category or type of book defined by its content, style, or form",
		Example:     "Adventure",
		Output:      "string",
		Aliases:     []string{"type", "content", "style", "form", "literature", "classification"},
		Keywords:    []string{"book", "category", "defined", "adventure", "fiction", "non-fiction", "mystery", "romance", "sci-fi"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return bookGenre(f), nil
		},
	})
}
