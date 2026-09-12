package parity

import (
	"fmt"
	"slices"
	"strings"
	"time"

	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var fixtureLongName = strings.Repeat("a", 260) + "needle.txt"

var fixtureNow = time.Now().UTC()

type fixtureOption func(*search.Resource)

func withPath(path string) fixtureOption { return func(r *search.Resource) { r.Path = path } }
func withMime(mime string) fixtureOption { return func(r *search.Resource) { r.MimeType = mime } }
func withTitle(t string) fixtureOption   { return func(r *search.Resource) { r.Title = t } }
func withContent(c string) fixtureOption { return func(r *search.Resource) { r.Content = c } }
func withSize(s uint64) fixtureOption    { return func(r *search.Resource) { r.Size = s } }
func withMtime(m string) fixtureOption {
	return func(r *search.Resource) {
		t, err := time.Parse(time.RFC3339Nano, m)
		if err != nil {
			panic(err)
		}
		r.Mtime = &t
	}
}
func withID(id string) fixtureOption     { return func(r *search.Resource) { r.ID = id } }
func withParent(id string) fixtureOption { return func(r *search.Resource) { r.ParentID = id } }
func withRoot(id string) fixtureOption   { return func(r *search.Resource) { r.RootID = id } }
func isHidden() fixtureOption            { return func(r *search.Resource) { r.Hidden = true } }
func isDeleted() fixtureOption           { return func(r *search.Resource) { r.Deleted = true } }
func withTags(tags ...string) fixtureOption {
	return func(r *search.Resource) { r.Tags = tags }
}

func withFavorite(userID string) fixtureOption {
	return func(r *search.Resource) { r.Favorites = []string{userID} }
}

func withAudio(audio *libregraph.Audio) fixtureOption {
	return func(r *search.Resource) { r.Audio = audio }
}

func withPhoto(photo *libregraph.Photo) fixtureOption {
	return func(r *search.Resource) { r.Photo = photo }
}

func withLocation(location *libregraph.GeoCoordinates) fixtureOption {
	return func(r *search.Resource) { r.Location = location }
}

func fixtureDoc(name string, opts ...fixtureOption) search.Resource {
	mtime := fixtureNow
	r := search.Resource{
		ID:       "1$1!" + name,
		RootID:   "1$1!1",
		ParentID: "1$1!1",
		Path:     "./" + name,
		Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
		Document: content.Document{
			Name:     name,
			MimeType: "text/plain",
			Mtime:    &mtime,
			Size:     1000,
		},
	}

	for _, opt := range opts {
		opt(&r)
	}

	return r
}

func fixtureFolder(name string, opts ...fixtureOption) search.Resource {
	folder := []fixtureOption{withMime("httpd/unix-directory")}
	r := fixtureDoc(name, append(folder, opts...)...)
	r.Type = uint64(sprovider.ResourceType_RESOURCE_TYPE_CONTAINER)

	return r
}

func fixtureBulk(count int) []search.Resource {
	docs := make([]search.Resource, 0, count)
	for i := range count {
		docs = append(docs, fixtureDoc(fmt.Sprintf("bulk-%d.txt", i)))
	}

	return docs
}

func fixtureTree() (parent, child search.Resource) {
	parent = fixtureFolder("parent", withID("1$1!2"))
	child = fixtureDoc("child.pdf", withID("1$1!3"), withParent(parent.ID), withPath("./parent/child.pdf"))

	return parent, child
}

func treeIsLeft(names ...string) []expectation {
	left := func(name string) []string {
		if slices.Contains(names, name) {
			return []string{name}
		}

		return nil
	}

	return []expectation{
		{`name:"*parent*"`, left("parent")},
		{`name:"*child*"`, left("child.pdf")},
	}
}
