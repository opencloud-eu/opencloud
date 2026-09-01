# Engine parity

Written by the parity suite (`UPDATE_SEARCH_PARITY_MATRIX=true go test ./services/search/pkg/parity/`), do not edit.
Every case runs against bleve and OpenSearch. `same?` is ✅ when both answer as
expected, `❌ known` when an engine's divergence is documented in the case
(`engineOverrides`), `❌` when it is not. `✅ stale` when every engine
answers the expected value although the case still documents a
divergence, that override can come out.

## Queries

### name

Fixtures:

- `new-folder`, folder
- `quarterly notes.txt`
- `Report.txt`
- `Übung.txt`
- `a+b.txt`
- `c(d).txt`
- `e&f.txt`
- `v1.2.3.txt`
- `foo bar.txt`
- `aaaaaaaaaa...edle.txt`

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| NAME-01 | `new` | new-folder | new-folder | new-folder | ✅ |
| NAME-02 | `quarterly` | quarterly notes.txt | quarterly notes.txt | quarterly notes.txt | ✅ |
| NAME-03 | `report` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-04 | `name:"*new-folder*"` | new-folder | new-folder | new-folder | ✅ |
| NAME-05 | `name:"*w-fol*"` | new-folder | new-folder | new-folder | ✅ |
| NAME-06 | `name:"*oo ba*"` | foo bar.txt | foo bar.txt | foo bar.txt | ✅ |
| NAME-07 | `name:"*REPORT*"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-08 | `name:"*übung*"` | Übung.txt | Übung.txt | Übung.txt | ✅ |
| NAME-09 | `name:"*ÜBUNG*"` | Übung.txt | Übung.txt | Übung.txt | ✅ |
| NAME-10 | `name:"*a+b*"` | a+b.txt | a+b.txt | a+b.txt | ✅ |
| NAME-11 | `name:"*c(d)*"` | c(d).txt | c(d).txt | c(d).txt | ✅ |
| NAME-12 | `name:"*e&f*"` | e&f.txt | e&f.txt | e&f.txt | ✅ |
| NAME-13 | `name:"*v1.2*"` | v1.2.3.txt | v1.2.3.txt | v1.2.3.txt | ✅ |
| NAME-14 | `new-folder` | new-folder | new-folder | new-folder | ✅ |
| NAME-15 | `*folder*` | new-folder | new-folder | new-folder | ✅ |
| NAME-16 | `name:"*foo bar*"` | foo bar.txt | foo bar.txt | foo bar.txt | ✅ |
| NAME-17 | `name:"foo bar.txt"` | foo bar.txt | foo bar.txt | foo bar.txt | ✅ |
| NAME-18 | `name:"*needle*"` | aaaaaaaaaa...edle.txt | aaaaaaaaaa...edle.txt | aaaaaaaaaa...edle.txt | ✅ |
| NAME-19 | `name:"report*"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-20 | `name:"*report"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-21 | `name:"Rep*rt.txt"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-22 | `Name:"*report*"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-23 | `NAME:"*report*"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-24 | `name:Rep?rt.txt` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-25 | `name:"*eport"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-26 | `name:"repor*"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-27 | `REPORT` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-28 | `name:REPORT` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-29 | `name:"REPORT.TXT"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-30 | `name:"FOO BAR.TXT"` | foo bar.txt | foo bar.txt | foo bar.txt | ✅ |
| NAME-31 | `name:"ÜBUNG.TXT"` | Übung.txt | Übung.txt | Übung.txt | ✅ |
| NAME-32 | `name:"folder*"` | no match | no match | no match | ✅ |
| NAME-33 | `name:"*new"` | no match | no match | no match | ✅ |
| NAME-34 | `name:new` | new-folder | new-folder | new-folder | ✅ |
| NAME-35 | `name:"new"` | new-folder | new-folder | new-folder | ✅ |
| NAME-36 | `name:"new-folder"` | new-folder | new-folder | new-folder | ✅ |
| NAME-37 | `name:"new-*"` | new-folder | new-folder | new-folder | ✅ |
| NAME-38 | `name:"new*"` | new-folder | new-folder | new-folder | ✅ |
| NAME-39 | `name:new-*` | new-folder | new-folder | new-folder | ✅ |
| NAME-40 | `name:"*-folder"` | new-folder | new-folder | new-folder | ✅ |
| NAME-41 | `name:"Rep?rt.txt"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-42 | `name="Report.txt"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-43 | `name="REPORT.TXT"` | Report.txt | Report.txt | Report.txt | ✅ |
| NAME-44 | `name="new"` | no match | no match | no match | ✅ |
| NAME-45 | `name="new-folder"` | new-folder | new-folder | new-folder | ✅ |
| NAME-46 | `name="foo bar.txt"` | foo bar.txt | foo bar.txt | foo bar.txt | ✅ |

### extension

Fixtures:

- `report.txt`
- `notes.md`, MimeType = text/markdown
- `archive`, folder

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| EXTENSION-01 | `txt` | report.txt | report.txt | report.txt | ✅ |
| EXTENSION-02 | `md` | notes.md | notes.md | notes.md | ✅ |
| EXTENSION-03 | `name:"*.txt"` | report.txt | report.txt | report.txt | ✅ |
| EXTENSION-04 | `report` | report.txt | report.txt | report.txt | ✅ |

### tags

Fixtures:

- `invoice.txt`, Tags = foo-bar
- `memo.txt`, Tags = foo
- `spaced.txt`, Tags = spaced tag
- `project`, folder, Tags = work
- `draft.txt`, Path = ./project/draft.txt
- `longtag.txt`, Tags = zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzneedle

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| TAGS-01 | `name:"*foo-bar*"` | no match | no match | no match | ✅ |
| TAGS-02 | `tag:("foo-bar")` | invoice.txt | invoice.txt | invoice.txt | ✅ |
| TAGS-03 | `tag:("foo")` | memo.txt | memo.txt | memo.txt | ✅ |
| TAGS-04 | `tag:("FOO-BAR")` | invoice.txt | invoice.txt | invoice.txt | ✅ |
| TAGS-05 | `tag:("*foo*")` | invoice.txt, memo.txt | invoice.txt, memo.txt | invoice.txt, memo.txt | ✅ |
| TAGS-06 | `tag:("spaced tag")` | spaced.txt | spaced.txt | spaced.txt | ✅ |
| TAGS-07 | `tag:("*paced ta*")` | spaced.txt | spaced.txt | spaced.txt | ✅ |
| TAGS-08 | `tag:("work")` | project | project | project | ✅ |
| TAGS-09 | `tag:("zzzzzzzzzzzzzzzzzzzzzzzzzz...zzzzzzzzzzzzzzzzneedle")` | longtag.txt | longtag.txt | longtag.txt | ✅ |

### title

Fixtures:

- `q1.html`, MimeType = text/html, Title = "quarterly report"

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| TITLE-01 | `Title:"quarterly report"` | q1.html | q1.html | q1.html | ✅ |
| TITLE-02 | `Title:quarterly` | q1.html | q1.html | q1.html | ✅ |
| TITLE-03 | `Title:QUARTERLY` | q1.html | q1.html | q1.html | ✅ |
| TITLE-04 | `Title:quarterl*` | q1.html | q1.html | q1.html | ✅ |
| TITLE-05 | `Title:"*ly rep*"` | q1.html | q1.html | q1.html | ✅ |
| TITLE-06 | `title:quarterly` | q1.html | q1.html | q1.html | ✅ |
| TITLE-07 | `Title:"QUARTERLY REPORT"` | q1.html | q1.html | q1.html | ✅ |
| TITLE-08 | `Title="quarterly report"` | q1.html | q1.html | q1.html | ✅ |
| TITLE-09 | `Title="quarterly"` | no match | no match | no match | ✅ |

### content

Fixtures:

- `monthly.txt`, Content = "the monthly reports are due"
- `links.txt`, Content = "see https://opencloud.example.com/help or write to alan@example.org"

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| CONTENT-01 | `Content:report` | no match | no match | no match | ✅ |
| CONTENT-02 | `Content:REPORTS` | monthly.txt | monthly.txt | monthly.txt | ✅ |
| CONTENT-03 | `Content:"monthly reports"` | monthly.txt | monthly.txt | monthly.txt | ✅ |
| CONTENT-04 | `Content:"reports monthly"` | no match | no match | no match | ✅ |
| CONTENT-05 | `Content:report*` | monthly.txt | monthly.txt | monthly.txt | ✅ |
| CONTENT-06 | `Content:*eport*` | monthly.txt | monthly.txt | monthly.txt | ✅ |
| CONTENT-07 | `Content:month*` | monthly.txt | monthly.txt | monthly.txt | ✅ |
| CONTENT-08 | `Content:"https://opencloud.example.com/help"` | links.txt | links.txt | links.txt | ✅ |
| CONTENT-09 | `Content:"alan@example.org"` | links.txt | links.txt | links.txt | ✅ |
| CONTENT-10 | `Content:opencloud` | links.txt | links.txt | links.txt | ✅ |

### cjk

Fixtures:

- `报告.txt`, Content = "这是一个关于年度销售的中文文档"
- `说明书.pdf`, MimeType = application/pdf, Content = "产品说明与安装步骤"
- `手册.txt`, Content = "学生手册"
- `图片`, folder
- `english.txt`, Content = "annual sales report"

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| CJK-01 | `Content:中文` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-02 | `Content:中文文档` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-03 | `Content:"中文文档"` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-04 | `Content:销售` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-05 | `Content:说明` | 说明书.pdf | 说明书.pdf | 说明书.pdf | ✅ |
| CJK-06 | `name:报告` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-07 | `name:"*报告*"` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-08 | `name:"报告.txt"` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-09 | `报告` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-10 | `说明书` | 说明书.pdf | 说明书.pdf | 说明书.pdf | ✅ |
| CJK-11 | `name:图片` | 图片 | 图片 | 图片 | ✅ |
| CJK-12 | `name:"*图片*"` | 图片 | 图片 | 图片 | ✅ |
| CJK-13 | `图片` | 图片 | 图片 | 图片 | ✅ |
| CJK-14 | `Content:学生` | 手册.txt | 手册.txt | 手册.txt | ✅ |
| CJK-15 | `Content:手册` | 手册.txt | 手册.txt | 手册.txt | ✅ |
| CJK-16 | `Content:生手` | 手册.txt | 手册.txt | 手册.txt | ✅ |
| CJK-17 | `name:"报*"` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-18 | `name:"*告*"` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-19 | `name:"*册*"` | 手册.txt | 手册.txt | 手册.txt | ✅ |
| CJK-20 | `name:"图?"` | 图片 | 图片 | 图片 | ✅ |
| CJK-21 | `name:"报?.txt"` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-22 | `name:"*报?*"` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-23 | `Content:中文*` | no match | no match | no match | ✅ |
| CJK-24 | `Content:*销售*` | no match | no match | no match | ✅ |
| CJK-25 | `Content:*文档` | no match | no match | no match | ✅ |
| CJK-26 | `Content:文*` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-27 | `Content:*文*` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-28 | `Content:*文` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-29 | `Content:销*` | 报告.txt | 报告.txt | 报告.txt | ✅ |
| CJK-30 | `Content:说*` | 说明书.pdf | 说明书.pdf | 说明书.pdf | ✅ |
| CJK-31 | `Content:*明` | 说明书.pdf | 说明书.pdf | 说明书.pdf | ✅ |

### favorites

Fixtures:

- `starred.txt`, Favorites = A1B2-Upper
- `plain.txt`
- `keepsakes`, folder, Favorites = A1B2-Upper
- `photo.jpg`, MimeType = image/jpeg, Path = ./keepsakes/photo.jpg

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| FAVORITES-01 | `Favorites:"A1B2-Upper"` | keepsakes, starred.txt | keepsakes, starred.txt | keepsakes, starred.txt | ✅ |
| FAVORITES-02 | `favorite:"A1B2-Upper"` | keepsakes, starred.txt | keepsakes, starred.txt | keepsakes, starred.txt | ✅ |
| FAVORITES-03 | `Favorites:"somebody-else"` | no match | no match | no match | ✅ |

### mediatype

Fixtures:

- `notes.md`, MimeType = text/markdown
- `photo.jpg`, MimeType = image/jpeg
- `albums`, folder
- `drafts`, folder

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| MEDIATYPE-01 | `mediatype:text/markdown` | notes.md | notes.md | notes.md | ✅ |
| MEDIATYPE-02 | `mediatype:TEXT/MARKDOWN` | notes.md | notes.md | notes.md | ✅ |
| MEDIATYPE-03 | `mediatype:image/jpeg` | photo.jpg | photo.jpg | photo.jpg | ✅ |
| MEDIATYPE-04 | `mediatype:*jpeg` | photo.jpg | photo.jpg | photo.jpg | ✅ |
| MEDIATYPE-05 | `mediatype:image` | photo.jpg | photo.jpg | photo.jpg | ✅ |
| MEDIATYPE-06 | `mediatype:folder` | albums, drafts | albums, drafts | albums, drafts | ✅ |
| MEDIATYPE-07 | `mediatype:file` | notes.md, photo.jpg | notes.md, photo.jpg | notes.md, photo.jpg | ✅ |
| MEDIATYPE-08 | `NOT mediatype:file` | albums, drafts | albums, drafts | albums, drafts | ✅ |
| MEDIATYPE-09 | `mediatype:file OR mediatype:image` | notes.md, photo.jpg | notes.md, photo.jpg | notes.md, photo.jpg | ✅ |
| MEDIATYPE-10 | `NOT mediatype:(image OR folder)` | notes.md | notes.md | notes.md | ✅ |

### path

Fixtures:

- `parent`, folder
- `child.jpg`, MimeType = image/jpeg, Path = ./parent/child.jpg
- `docs-lower`, folder, Path = ./documents
- `docs-upper`, folder, Path = ./DOCUMENTS
- `docs-mixed`, folder, Path = ./Documents

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| PATH-01 | `path:"./parent"` | child.jpg, parent | child.jpg, parent | child.jpg, parent | ✅ |
| PATH-02 | `path:"./parent/child.jpg"` | child.jpg | child.jpg | child.jpg | ✅ |
| PATH-03 | `path:"./Parent"` | no match | no match | no match | ✅ |
| PATH-04 | `path:"*child*"` | child.jpg | child.jpg | child.jpg | ✅ |
| PATH-05 | `path:"./documents"` | docs-lower | docs-lower | docs-lower | ✅ |
| PATH-06 | `path:"./DOCUMENTS"` | docs-upper | docs-upper | docs-upper | ✅ |
| PATH-07 | `path:"./Documents"` | docs-mixed | docs-mixed | docs-mixed | ✅ |
| PATH-08 | `path:"./parent/"` | child.jpg, parent | child.jpg, parent | child.jpg, parent | ✅ |

### fields

Fixtures:

- `small.txt`, ID = 1$1!small.txt, Size = 42
- `old.txt`, ID = 1$1!old.txt, Mtime = 2020-01-01T00:00:00Z
- `known.txt`, ID = 1$1!23
- `cased.txt`, ID = 1$1!AB-23
- `hidden.txt`, ID = 1$1!hidden.txt, hidden
- `plain.txt`, ID = 1$1!plain.txt
- `box`, ID = 1$1!box, folder
- `boxed.txt`, ID = 1$1!boxed.txt, Path = ./box/boxed.txt
- `song.mp3`, ID = 1$1!song.mp3, MimeType = audio/mpeg

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| FIELDS-01 | `size:42` | small.txt | small.txt | small.txt | ✅ |
| FIELDS-02 | `mtime<"2021-01-01T00:00:00Z"` | old.txt | old.txt | old.txt | ✅ |
| FIELDS-03 | `id:"1$1!23"` | known.txt | known.txt | known.txt | ✅ |
| FIELDS-04 | `hidden:true` | hidden.txt | hidden.txt | hidden.txt | ✅ |
| FIELDS-05 | `type:file` | boxed.txt, cased.txt, hidden.txt, known.txt, old.txt, plain.txt, small.txt, song.mp3 | boxed.txt, cased.txt, hidden.txt, known.txt, old.txt, plain.txt, small.txt, song.mp3 | boxed.txt, cased.txt, hidden.txt, known.txt, old.txt, plain.txt, small.txt, song.mp3 | ✅ |
| FIELDS-06 | `type:folder` | box | box | box | ✅ |
| FIELDS-07 | `unknown:field` | no match | no match | no match | ✅ |
| FIELDS-08 | `type:File` | boxed.txt, cased.txt, hidden.txt, known.txt, old.txt, plain.txt, small.txt, song.mp3 | boxed.txt, cased.txt, hidden.txt, known.txt, old.txt, plain.txt, small.txt, song.mp3 | boxed.txt, cased.txt, hidden.txt, known.txt, old.txt, plain.txt, small.txt, song.mp3 | ✅ |
| FIELDS-09 | `type:FOLDER` | box | box | box | ✅ |
| FIELDS-10 | `hidden:TRUE` | hidden.txt | hidden.txt | hidden.txt | ✅ |
| FIELDS-11 | `id:"1$1!AB-23"` | cased.txt | cased.txt | cased.txt | ✅ |
| FIELDS-12 | `id:"1$1!ab-23"` | no match | no match | no match | ✅ |
| FIELDS-13 | `audio.artist:"Some Artist"` | song.mp3 | song.mp3 | song.mp3 | ✅ |
| FIELDS-14 | `audio.artist:"some artist"` | song.mp3 | song.mp3 | song.mp3 | ✅ |
| FIELDS-15 | `audio.duration>100` | no match | no match | no match | ✅ |

### deleted

Fixtures:

- `trashed.txt`, deleted
- `kept.txt`
- `bin`, folder, deleted
- `receipt.txt`, Path = ./bin/receipt.txt, deleted
- `shelf`, folder
- `book.txt`, Path = ./shelf/book.txt

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| DELETED-01 | `name:"*trashed*"` | no match | no match | no match | ✅ |
| DELETED-02 | `name:"*.txt"` | book.txt, kept.txt | book.txt, kept.txt | book.txt, kept.txt | ✅ |
| DELETED-03 | `name:"*receipt*"` | no match | no match | no match | ✅ |
| DELETED-04 | `path:"./bin"` | no match | no match | no match | ✅ |
| DELETED-05 | `path:"./shelf"` | book.txt, shelf | book.txt, shelf | book.txt, shelf | ✅ |

### visibility

Fixtures:

- `visible.txt`
- `dotfile.txt`, hidden
- `.private`, folder, hidden
- `secret.txt`, Path = ./.private/secret.txt, hidden

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| VISIBILITY-01 | `hidden:true` | .private, dotfile.txt, secret.txt | .private, dotfile.txt, secret.txt | .private, dotfile.txt, secret.txt | ✅ |
| VISIBILITY-02 | `hidden:TRUE` | .private, dotfile.txt, secret.txt | .private, dotfile.txt, secret.txt | .private, dotfile.txt, secret.txt | ✅ |
| VISIBILITY-03 | `hidden:false` | visible.txt | visible.txt | visible.txt | ✅ |
| VISIBILITY-04 | `name:"*secret*"` | secret.txt | secret.txt | secret.txt | ✅ |
| VISIBILITY-05 | `path:"./.private"` | .private, secret.txt | .private, secret.txt | .private, secret.txt | ✅ |
| VISIBILITY-06 | `hidden:banana` | no match | no match | no match | ✅ |
| VISIBILITY-07 | `hidden:"true"` | .private, dotfile.txt, secret.txt | .private, dotfile.txt, secret.txt | .private, dotfile.txt, secret.txt | ✅ |

### boolean

Fixtures:

- `alpha.txt`, Tags = red
- `beta.txt`, Tags = blue
- `gamma.md`, MimeType = text/markdown

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| BOOLEAN-01 | `name:"*alpha*" AND name:"*txt*"` | alpha.txt | alpha.txt | alpha.txt | ✅ |
| BOOLEAN-02 | `name:"*alpha*" OR name:"*beta*"` | alpha.txt, beta.txt | alpha.txt, beta.txt | alpha.txt, beta.txt | ✅ |
| BOOLEAN-03 | `name:"*a*" AND NOT name:"*alpha*"` | beta.txt, gamma.md | beta.txt, gamma.md | beta.txt, gamma.md | ✅ |
| BOOLEAN-04 | `name:"*a*" AND tag:("red")` | alpha.txt | alpha.txt | alpha.txt | ✅ |
| BOOLEAN-05 | `(name:"*alpha*" OR name:"*beta*") AND mediatype:text/plain` | alpha.txt, beta.txt | alpha.txt, beta.txt | alpha.txt, beta.txt | ✅ |
| BOOLEAN-06 | `name:"*a*" AND mediatype:text/markdown` | gamma.md | gamma.md | gamma.md | ✅ |

### samename

Fixtures:

- `doc`, folder
- `doc.pdf`
- `file.pdf`
- `doc.pdf`, Path = ./doc/doc.pdf
- `file.pdf`, Path = ./doc/file.pdf

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| SAMENAME-01 | `name:"*doc*"` | 3 items | doc, doc.pdf, doc.pdf | doc, doc.pdf, doc.pdf | ✅ |
| SAMENAME-02 | `name:"*doc*"` in 1$1!1 under ./doc | 2 items | doc, doc.pdf | doc, doc.pdf | ✅ |
| SAMENAME-03 | `name:"*file*"` | 2 items | file.pdf, file.pdf | file.pdf, file.pdf | ✅ |
| SAMENAME-04 | `name:"*file*"` in 1$1!1 under ./doc | 1 items | file.pdf | file.pdf | ✅ |

### stress

Fixtures:

- `quarterly report.docx`, MimeType = application/vnd.openxmlformats-officedocument.wordprocessingml.document, Tags = final, Size = 2000
- `draft report.txt`, Tags = draft, Size = 50, Mtime = 2020-01-01T00:00:00Z
- `photo.jpg`, MimeType = image/jpeg, Tags = final
- `notes.md`, MimeType = text/markdown, hidden
- `archive`, folder

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| STRESS-01 | `name:"*report*" AND mediatype:document` | draft report.txt, quarterly report.docx | draft report.txt, quarterly report.docx | draft report.txt, quarterly report.docx | ✅ |
| STRESS-02 | `name:"*report*" AND NOT tag:("draft")` | quarterly report.docx | quarterly report.docx | quarterly report.docx | ✅ |
| STRESS-03 | `(tag:("final") OR tag:("draft")) AND mediatype:image` | photo.jpg | photo.jpg | photo.jpg | ✅ |
| STRESS-04 | `mediatype:document AND mtime>"2021-01-01T00:00:00Z"` | notes.md, quarterly report.docx | notes.md, quarterly report.docx | notes.md, quarterly report.docx | ✅ |
| STRESS-05 | `name:"*report*" AND size>100` | quarterly report.docx | quarterly report.docx | quarterly report.docx | ✅ |
| STRESS-06 | `tag:("final") AND NOT mediatype:folder` | photo.jpg, quarterly report.docx | photo.jpg, quarterly report.docx | photo.jpg, quarterly report.docx | ✅ |
| STRESS-07 | `hidden:true AND name:"*notes*"` | notes.md | notes.md | notes.md | ✅ |
| STRESS-08 | `name:quarterly report` | quarterly report.docx | quarterly report.docx | quarterly report.docx | ✅ |
| STRESS-09 | `name:"quarterly report"` | quarterly report.docx | quarterly report.docx | quarterly report.docx | ✅ |
| STRESS-10 | `name:"quarterly report.docx"` | quarterly report.docx | quarterly report.docx | quarterly report.docx | ✅ |
| STRESS-11 | `NOT tag:("draft")` | archive, notes.md, photo.jpg, quarterly report.docx | archive, notes.md, photo.jpg, quarterly report.docx | archive, notes.md, photo.jpg, quarterly report.docx | ✅ |
| STRESS-12 | `tag:("final") OR hidden:true` | notes.md, photo.jpg, quarterly report.docx | notes.md, photo.jpg, quarterly report.docx | notes.md, photo.jpg, quarterly report.docx | ✅ |
| STRESS-13 | `(name:"*report*" OR name:"*notes..."draft") OR hidden:true)` | quarterly report.docx | quarterly report.docx | quarterly report.docx | ✅ |
| STRESS-14 | `mediatype:image OR (mediatype:document AND tag:("draft"))` | draft report.txt, photo.jpg | draft report.txt, photo.jpg | draft report.txt, photo.jpg | ✅ |
| STRESS-15 | `NOT (mediatype:folder OR hidden:true)` | draft report.txt, photo.jpg, quarterly report.docx | draft report.txt, photo.jpg, quarterly report.docx | draft report.txt, photo.jpg, quarterly report.docx | ✅ |
| STRESS-16 | `name:"*report*" AND (size>100 OR tag:("draft"))` | draft report.txt, quarterly report.docx | draft report.txt, quarterly report.docx | draft report.txt, quarterly report.docx | ✅ |

### everything

Fixtures:

- `alpha.txt`
- `beta.txt`
- `box`, folder

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| EVERYTHING-01 | `*` | alpha.txt, beta.txt, box | alpha.txt, beta.txt, box | alpha.txt, beta.txt, box | ✅ |
| EVERYTHING-02 | `name:"*"` | alpha.txt, beta.txt, box | alpha.txt, beta.txt, box | alpha.txt, beta.txt, box | ✅ |
| EVERYTHING-03 | `*` with a limit of 2 | 2 items | alpha.txt, beta.txt | alpha.txt, beta.txt | ✅ |
| EVERYTHING-04 | `*` with a limit of -1 | 3 items | alpha.txt, beta.txt, box | alpha.txt, beta.txt, box | ✅ |

### range

Fixtures:

- `small.txt`, Size = 50
- `big.txt`, Size = 500
- `ancient.txt`, Size = 10, Mtime = 2020-01-01T00:00:00Z

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| RANGE-01 | `size>100` | big.txt | big.txt | big.txt | ✅ |
| RANGE-02 | `size<100` | ancient.txt, small.txt | ancient.txt, small.txt | ancient.txt, small.txt | ✅ |
| RANGE-03 | `mtime>"2021-01-01T00:00:00Z"` | big.txt, small.txt | big.txt, small.txt | big.txt, small.txt | ✅ |
| RANGE-04 | `mtime<"2021-01-01T00:00:00Z"` | ancient.txt | ancient.txt | ancient.txt | ✅ |
| RANGE-05 | `Mtime:"today"` | big.txt, small.txt | big.txt, small.txt | big.txt, small.txt | ✅ |
| RANGE-06 | `Mtime:"yesterday"` | no match | no match | no match | ✅ |
| RANGE-07 | `mtime>2021` | no match | no match | no match | ✅ |
| RANGE-08 | `name>100` | no match | no match | no match | ✅ |

### scope

Fixtures:

- `parent`, ID = 1$1!parent, folder
- `child.pdf`, ID = 1$1!child.pdf, Path = ./parent/child.pdf
- `outside.txt`, ID = 1$1!outside.txt
- `elsewhere.txt`, ID = 1$1!elsewhere.txt

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| SCOPE-01 | `*` | child.pdf, elsewhere.txt, outside.txt, parent | child.pdf, elsewhere.txt, outside.txt, parent | child.pdf, elsewhere.txt, outside.txt, parent | ✅ |
| SCOPE-02 | `*` in 1$1!1 | child.pdf, outside.txt, parent | child.pdf, outside.txt, parent | child.pdf, outside.txt, parent | ✅ |
| SCOPE-03 | `*` in 1$1!1 under ./parent | child.pdf, parent | child.pdf, parent | child.pdf, parent | ✅ |
| SCOPE-04 | `*` in 1$1!1 under ./parent/child.pdf | child.pdf | child.pdf | child.pdf | ✅ |
| SCOPE-05 | `name:"*elsewhere*"` in 1$1!1 | no match | no match | no match | ✅ |
| SCOPE-06 | `driveid:"2$2"` | elsewhere.txt | elsewhere.txt | elsewhere.txt | ✅ |
| SCOPE-07 | `driveid:"1$1"` | child.pdf, outside.txt, parent | child.pdf, outside.txt, parent | child.pdf, outside.txt, parent | ✅ |
| SCOPE-08 | `NOT driveid:"2$2"` | child.pdf, outside.txt, parent | child.pdf, outside.txt, parent | child.pdf, outside.txt, parent | ✅ |
| SCOPE-09 | `name:"*elsewhere*" AND driveid:"2$2"` | elsewhere.txt | elsewhere.txt | elsewhere.txt | ✅ |

### invalid

Fixtures:

- `alpha.txt`

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| INVALID-01 | `AND mediatype:document` | bad request | bad request | bad request | ✅ |
| INVALID-02 | `mediatype:document AND` | alpha.txt | alpha.txt | alpha.txt | ✅ |

## Operations

### delete

Fixtures:

- `parent`, ID = 1$1!2, folder
- `child.pdf`, ID = 1$1!3, Path = ./parent/child.pdf

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| DELETE-01 | takes the resource out of the results, then `name:"*parent*"` | parent | parent | parent | ✅ |
| DELETE-01 | takes the resource out of the results, then `name:"*child*"` | no match | no match | no match | ✅ |
| DELETE-02 | takes the descendants along, then `name:"*parent*"` | no match | no match | no match | ✅ |
| DELETE-02 | takes the descendants along, then `name:"*child*"` | no match | no match | no match | ✅ |
| DELETE-03 | leaves the resource in the index, then `DocCount()` | 2 | 2 | 2 | ✅ |
| DELETE-04 | takes a resource out that was just written, then `name:"*fresh*"` | no match | no match | no match | ✅ |

### restore

Fixtures:

- `parent`, ID = 1$1!2, folder
- `child.pdf`, ID = 1$1!3, Path = ./parent/child.pdf
- `file.txt`, ID = 1$1!5, Path = ./.secret/file.txt, hidden

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| RESTORE-01 | brings the descendants back, then `name:"*parent*"` | parent | parent | parent | ✅ |
| RESTORE-01 | brings the descendants back, then `name:"*child*"` | child.pdf | child.pdf | child.pdf | ✅ |
| RESTORE-02 | leaves the hidden flag alone, then `hidden:true` | file.txt | file.txt | file.txt | ✅ |

### purge

Fixtures:

- `parent`, ID = 1$1!2, folder
- `child.pdf`, ID = 1$1!3, Path = ./parent/child.pdf

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| PURGE-01 | removes one resource, then `name:"*parent*"` | parent | parent | parent | ✅ |
| PURGE-01 | removes one resource, then `name:"*child*"` | no match | no match | no match | ✅ |
| PURGE-01 | removes one resource, then `DocCount()` | 1 | 1 | 1 | ✅ |
| PURGE-02 | removes the tree, then `name:"*parent*"` | no match | no match | no match | ✅ |
| PURGE-02 | removes the tree, then `name:"*child*"` | no match | no match | no match | ✅ |
| PURGE-02 | removes the tree, then `DocCount()` | 0 | 0 | 0 | ✅ |
| PURGE-03 | takes only the deleted ones when it is told to, then `name:"*parent*"` | parent | parent | parent | ✅ |
| PURGE-03 | takes only the deleted ones when it is told to, then `name:"*child*"` | no match | no match | no match | ✅ |

### purgespace

Fixtures:

- `inSpace.txt`, ID = 1$1!4
- `elsewhere.txt`, ID = 2$2!4
- `bulk-0.txt`, ID = 1$1!bulk-0.txt
- `bulk-1.txt`, ID = 1$1!bulk-1.txt
- `bulk-2.txt`, ID = 1$1!bulk-2.txt
- `bulk-3.txt`, ID = 1$1!bulk-3.txt
- `bulk-4.txt`, ID = 1$1!bulk-4.txt
- `bulk-5.txt`, ID = 1$1!bulk-5.txt
- `bulk-6.txt`, ID = 1$1!bulk-6.txt
- `bulk-7.txt`, ID = 1$1!bulk-7.txt
- ... and 52 more of the same

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| PURGESPACE-01 | leaves the other space alone, then `name:"*inSpace*"` | no match | no match | no match | ✅ |
| PURGESPACE-01 | leaves the other space alone, then `name:"*elsewhere*"` | elsewhere.txt | elsewhere.txt | elsewhere.txt | ✅ |
| PURGESPACE-02 | takes a space out that holds more than one round, then `name:"*bulk*"` | no match | no match | no match | ✅ |
| PURGESPACE-02 | takes a space out that holds more than one round, then `name:"*elsewhere*"` | elsewhere.txt | elsewhere.txt | elsewhere.txt | ✅ |
| PURGESPACE-02 | takes a space out that holds more than one round, then `DocCount()` | 1 | 1 | 1 | ✅ |

### move

Fixtures:

- `parent`, ID = 1$1!2, folder
- `child.pdf`, ID = 1$1!3, Path = ./parent/child.pdf

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| MOVE-01 | carries the descendants to the new path, then `path:"./my/newname/child.pdf"` | child.pdf | child.pdf | child.pdf | ✅ |
| MOVE-01 | carries the descendants to the new path, then `path:"./parent/child.pdf"` | no match | no match | no match | ✅ |
| MOVE-02 | through the trash and back leaves the flag behind, then `hidden:true` | no match | no match | no match | ✅ |

### rootscope

Fixtures:

- `target.txt`, ID = 1$1!3, Path = ./same/path.txt
- `twin.txt`, ID = 2$2!3, Path = ./same/path.txt
- `target.txt`, ID = 1$1!3, Path = ./same/path.txt, deleted
- `twin.txt`, ID = 2$2!3, Path = ./same/path.txt, deleted

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| ROOTSCOPE-01 | deletes only the one in the target root, then `name:"*target*"` | no match | no match | no match | ✅ |
| ROOTSCOPE-01 | deletes only the one in the target root, then `name:"*twin*"` | twin.txt | twin.txt | twin.txt | ✅ |
| ROOTSCOPE-02 | restores only the one in the target root, then `name:"*target*"` | target.txt | target.txt | target.txt | ✅ |
| ROOTSCOPE-02 | restores only the one in the target root, then `name:"*twin*"` | no match | no match | no match | ✅ |
| ROOTSCOPE-04 | purges only the one in the target root, then `name:"*target*"` | no match | no match | no match | ✅ |
| ROOTSCOPE-04 | purges only the one in the target root, then `name:"*twin*"` | twin.txt | twin.txt | twin.txt | ✅ |
| ROOTSCOPE-03 | moves only the one in the target root, then `path:"./moved.txt"` | moved.txt | moved.txt | moved.txt | ✅ |
| ROOTSCOPE-03 | moves only the one in the target root, then `path:"./same/path.txt"` | twin.txt | twin.txt | twin.txt | ✅ |

### casepath

Fixtures:

- `Documents`, ID = 1$1!2, folder
- `Picture.jpg`, ID = 1$1!3, MimeType = image/jpeg, Path = ./Documents/Picture.jpg

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| CASEPATH-01 | takes the descendants along when deleting, then `path:"./Documents"` | no match | no match | no match | ✅ |
| CASEPATH-02 | takes the descendants along when moving, then `path:"./Other Documents"` | Other Documents, Picture.jpg | Other Documents, Picture.jpg | Other Documents, Picture.jpg | ✅ |
| CASEPATH-02 | takes the descendants along when moving, then `path:"./Documents"` | no match | no match | no match | ✅ |
| CASEPATH-03 | reaches the descendants when purging, then `path:"./Documents"` | no match | no match | no match | ✅ |
| CASEPATH-03 | reaches the descendants when purging, then `DocCount()` | 0 | 0 | 0 | ✅ |

### hidden

Fixtures:

- `parent`, ID = 1$1!2, folder
- `child.pdf`, ID = 1$1!3, Path = ./parent/child.pdf
- `parent`, ID = 1$1!2, folder, Path = ./.trash/parent, hidden
- `child.pdf`, ID = 1$1!3, Path = ./.trash/parent/child.pdf, hidden
- `parent`, ID = 1$1!2, folder, Path = ./.parent, hidden
- `child.pdf`, ID = 1$1!3, Path = ./.parent/child.pdf, hidden

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| HIDDEN-01 | follows a move into a dot folder, then `hidden:true` | child.pdf, parent | child.pdf, parent | child.pdf, parent | ✅ |
| HIDDEN-02 | follows a move into a plain folder, then `hidden:true` | no match | no match | no match | ✅ |
| HIDDEN-03 | follows a move renamed with a leading dot, then `hidden:true` | .parent, child.pdf | .parent, child.pdf | .parent, child.pdf | ✅ |
| HIDDEN-04 | follows a move out of a dot folder, then `hidden:true` | no match | no match | no match | ✅ |
| HIDDEN-05 | follows a move renamed without the leading dot, then `hidden:true` | no match | no match | no match | ✅ |
| HIDDEN-06 | follows a move within the same dot folder, then `hidden:true` | child.pdf, moved | child.pdf, moved | child.pdf, moved | ✅ |

### upsert

Fixtures:

- `parent`, ID = 1$1!2, folder
- `child.pdf`, ID = 1$1!3, Path = ./parent/child.pdf

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| UPSERT-01 | replaces the resource it already knows, then `name:"*parent*"` | parent | parent | parent | ✅ |
| UPSERT-01 | replaces the resource it already knows, then `name:"*child*"` | no match | no match | no match | ✅ |
| UPSERT-01 | replaces the resource it already knows, then `name:"*renamed*"` | renamed.pdf | renamed.pdf | renamed.pdf | ✅ |
| UPSERT-01 | replaces the resource it already knows, then `DocCount()` | 2 | 2 | 2 | ✅ |

### idempotency

Fixtures:

- `parent`, ID = 1$1!2, folder
- `child.pdf`, ID = 1$1!3, Path = ./parent/child.pdf

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| IDEMPOTENCY-01 | deleting the same resource twice is not an error, then `name:"*parent*"` | parent | parent | parent | ✅ |
| IDEMPOTENCY-01 | deleting the same resource twice is not an error, then `name:"*child*"` | no match | no match | no match | ✅ |
| IDEMPOTENCY-02 | deleting a resource the index does not have reports it, then `name:"*parent*"` | parent | parent | parent | ✅ |
| IDEMPOTENCY-02 | deleting a resource the index does not have reports it, then `name:"*child*"` | child.pdf | child.pdf | child.pdf | ✅ |
| IDEMPOTENCY-03 | restoring a resource that was never deleted leaves it alone, then `name:"*parent*"` | parent | parent | parent | ✅ |
| IDEMPOTENCY-03 | restoring a resource that was never deleted leaves it alone, then `name:"*child*"` | child.pdf | child.pdf | child.pdf | ✅ |
| IDEMPOTENCY-04 | purging the same resource twice reports the second one, then `name:"*parent*"` | parent | parent | parent | ✅ |
| IDEMPOTENCY-04 | purging the same resource twice reports the second one, then `name:"*child*"` | no match | no match | no match | ✅ |
| IDEMPOTENCY-04 | purging the same resource twice reports the second one, then `DocCount()` | 1 | 1 | 1 | ✅ |
| IDEMPOTENCY-05 | purging a resource the index does not have reports it, then `name:"*parent*"` | parent | parent | parent | ✅ |
| IDEMPOTENCY-05 | purging a resource the index does not have reports it, then `name:"*child*"` | child.pdf | child.pdf | child.pdf | ✅ |
| IDEMPOTENCY-05 | purging a resource the index does not have reports it, then `DocCount()` | 2 | 2 | 2 | ✅ |
| IDEMPOTENCY-06 | moving a resource onto its own path leaves it where it is, then `path:"./parent/child.pdf"` | child.pdf | child.pdf | child.pdf | ✅ |
| IDEMPOTENCY-06 | moving a resource onto its own path leaves it where it is, then `name:"*parent*"` | parent | parent | parent | ✅ |
| IDEMPOTENCY-07 | purging a whole space that holds nothing is not an error, then `name:"*parent*"` | parent | parent | parent | ✅ |
| IDEMPOTENCY-07 | purging a whole space that holds nothing is not an error, then `name:"*child*"` | child.pdf | child.pdf | child.pdf | ✅ |
| IDEMPOTENCY-07 | purging a whole space that holds nothing is not an error, then `DocCount()` | 2 | 2 | 2 | ✅ |

### batch

Fixtures:

- `parent`, ID = 1$1!2, folder
- `child.pdf`, ID = 1$1!3, Path = ./parent/child.pdf

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| BATCH-01 | holds what it was given until it is pushed, then `name:"*added*"` | no match | no match | no match | ✅ |
| BATCH-02 | writes what it was given once it is pushed, then `name:"*added*"` | added.pdf | added.pdf | added.pdf | ✅ |
| BATCH-02 | writes what it was given once it is pushed, then `DocCount()` | 3 | 3 | 3 | ✅ |
| BATCH-03 | takes a resource out the same way a delete does, then `name:"*parent*"` | parent | parent | parent | ✅ |
| BATCH-03 | takes a resource out the same way a delete does, then `name:"*child*"` | no match | no match | no match | ✅ |
| BATCH-04 | moves a resource the same way a move does, then `path:"./my/newname/child.pdf"` | child.pdf | child.pdf | child.pdf | ✅ |
| BATCH-04 | moves a resource the same way a move does, then `path:"./parent/child.pdf"` | no match | no match | no match | ✅ |
| BATCH-05 | keeps what another batch holds out of its push, then `name:"*added*"` | added.pdf | added.pdf | added.pdf | ✅ |
| BATCH-05 | keeps what another batch holds out of its push, then `name:"*other*"` | no match | no match | no match | ✅ |

## Response

### entity

Fixtures:

- `parent`, ID = 1$1!2, folder
- `bar.pdf`, ID = 1$1!3, MimeType = application/pdf, Path = ./parent/bar.pdf, Size = 1234
- `notes.txt`, ID = 1$1!4, Content = "foo bar baz"

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| ENTITY-01 | `name:"bar.pdf"` reads `Ref.Path` | ./parent/bar.pdf | ./parent/bar.pdf | ./parent/bar.pdf | ✅ |
| ENTITY-02 | `name:"bar.pdf"` reads `Name` | bar.pdf | bar.pdf | bar.pdf | ✅ |
| ENTITY-03 | `name:"bar.pdf"` reads `Id` | 1$1!3 | 1$1!3 | 1$1!3 | ✅ |
| ENTITY-04 | `name:"bar.pdf"` reads `ParentId` | 1$1!2 | 1$1!2 | 1$1!2 | ✅ |
| ENTITY-05 | `name:"bar.pdf"` reads `Ref.ResourceId` | 1$1!1 | 1$1!1 | 1$1!1 | ✅ |
| ENTITY-06 | `name:"bar.pdf"` reads `Size` | 1234 | 1234 | 1234 | ✅ |
| ENTITY-07 | `name:"bar.pdf"` reads `Type` | 1 | 1 | 1 | ✅ |
| ENTITY-08 | `name:"bar.pdf"` reads `MimeType` | application/pdf | application/pdf | application/pdf | ✅ |
| ENTITY-09 | `name:"bar.pdf"` reads `Deleted` | false | false | false | ✅ |
| ENTITY-10 | `name:"bar.pdf"` reads `Score` | above zero | above zero | above zero | ✅ |
| ENTITY-11 | `path:"./parent"` reads `TotalMatches` | 2 | 2 | 2 | ✅ |
| ENTITY-12 | `name:"*notes*"` reads `Highlights` | "" | "" | "" | ✅ |
| ENTITY-13 | `content:bar` reads `Highlights` | foo <mark>bar</mark> baz | foo <mark>bar</mark> baz | foo <mark>bar</mark> baz | ✅ |
| ENTITY-14 | moved to another parent, then `name:"newname"` reads `ParentId` | 1$1!9 | 1$1!9 | 1$1!9 | ✅ |
| ENTITY-15 | moved to another parent, then `name:"bar.pdf"` reads `ParentId` | 1$1!2 | 1$1!2 | 1$1!2 | ✅ |
| ENTITY-16 | moved to another parent, then `name:"bar.pdf"` reads `Ref.Path` | ./somewher.../bar.pdf | ./somewher.../bar.pdf | ./somewher.../bar.pdf | ✅ |

### metadata

Fixtures:

- `some_song.mp3`, ID = 1$1!5, MimeType = audio/mpeg
- `team.jpg`, ID = 1$1!6, MimeType = image/jpeg

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| METADATA-01 | `*song*` reads `Audio` | all 16 fields unchanged | all 16 fields unchanged | all 16 fields unchanged | ✅ |
| METADATA-02 | `*team*` reads `Location` | all 3 fields unchanged | all 3 fields unchanged | all 3 fields unchanged | ✅ |
| METADATA-03 | `*team*` reads `Audio` | none | none | none | ✅ |

## Aggregations

### aggregations

Fixtures:

- `a.mp3`, MimeType = audio/mpeg
- `b.mp3`, MimeType = audio/mpeg
- `c.mp3`, MimeType = audio/mpeg
- `d.mp3`, MimeType = audio/mpeg
- `e.mp3`, MimeType = audio/mpeg
- `f.mp3`, MimeType = audio/mpeg
- `g.mp3`, MimeType = audio/mpeg
- `a.jpg`, MimeType = image/jpeg
- `b.jpg`, MimeType = image/jpeg
- `c.jpg`, MimeType = image/jpeg
- `d.jpg`, MimeType = image/jpeg

| Case | Query | expected | bleve | OpenSearch | same? |
|---|---|---|---|---|---|
| AGG-01 | `mediatype:audio` reads `term buckets on audio.artist` | audio.arti... Floyd=2, audio.arti...�rhead=3 | audio.arti... Floyd=2, audio.arti...�rhead=3 | audio.arti... Floyd=2, audio.arti...�rhead=3 | ✅ |
| AGG-02 | `mediatype:audio` reads `no aggregations requested` | no match | no match | no match | ✅ |
| AGG-03 | `mediatype:audio` reads `artist and album buckets in one request` | audio.albu...Spades=1, audio.album Bomber=2, audio.album The Wall=2, audio.arti... Floyd=2, audio.arti...�rhead=3 | audio.albu...Spades=1, audio.album Bomber=2, audio.album The Wall=2, audio.arti... Floyd=2, audio.arti...�rhead=3 | audio.albu...Spades=1, audio.album Bomber=2, audio.album The Wall=2, audio.arti... Floyd=2, audio.arti...�rhead=3 | ✅ |
| AGG-04 | `mediatype:audio` reads `audio.year buckets per decade` | audio.year 1970-1980=2, audio.year 1980-1990=1, audio.year 1990-2000=1, audio.year 2000-2010=3 | audio.year 1970-1980=2, audio.year 1980-1990=1, audio.year 1990-2000=1, audio.year 2000-2010=3 | audio.year 1970-1980=2, audio.year 1980-1990=1, audio.year 1990-2000=1, audio.year 2000-2010=3 | ✅ |
| AGG-05 | `mediatype:audio` reads `open-ended audio.year ranges` | audio.year -1990=3, audio.year 2000-=3 | audio.year -1990=3, audio.year 2000-=3 | audio.year -1990=3, audio.year 2000-=3 | ✅ |
| AGG-06 | `mediatype:audio` reads `top-level metrics on audio.year` | audio.year max=2009, audio.year min=1971, audio.year sum=13942, audio.year... count=7 | audio.year max=2009, audio.year min=1971, audio.year sum=13942, audio.year... count=7 | audio.year max=2009, audio.year min=1971, audio.year sum=13942, audio.year... count=7 | ✅ |
| AGG-07 | `mediatype:image` reads `photo.takenDateTime buckets per date range` | photo.take...-01-01=1, photo.take...-09-01=2, photo.take...00:00Z=2 | photo.take...-01-01=1, photo.take...-09-01=2, photo.take...00:00Z=2 | photo.take...-01-01=1, photo.take...-09-01=2, photo.take...00:00Z=2 | ✅ |
| AGG-08 | `mediatype:image` reads `open-ended date ranges` | photo.take...-01-01=3, photo.take...01-01-=1 | photo.take...-01-01=3, photo.take...01-01-=1 | photo.take...-01-01=3, photo.take...01-01-=1 | ✅ |
| AGG-09 | `mediatype:image` reads `malformed date range bound` | error | error | error | ✅ |
