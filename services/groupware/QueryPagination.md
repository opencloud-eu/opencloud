# Query Pagination

## Single JMAP Backend

Paginating queries, or just plain lists of objects without query filters
(that end up being queries as well, since those are the only JMAP operations that allow
specifying an offset ("position") and limit), are fairly simple to deal with if we are
only considering delegating those to a single JMAP backend, such as Stalwart.

In those cases, we can either make use of

* `position` and `limit`,
* or `anchor` and `anchorOffset`

Considering the following list of elements:

|   0 |   1 |   2 |   3 |   4 |   5 |   6 |   7 |
|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `a` | `b` | `c` | `d` | `e` | `f` | `g` | `h` |

### Using `position` and `limit`

The first option simply uses a numeric offset, e.g.

| Page | `position` | `limit` | results       |
|------|------------|---------|---------------|
| 1    | `0`        | `3`     | `a`, `b`, `c` |
| 2    | `3`        | `3`     | `d`, `e`, `f` |
| 3    | `6`        | `3`     | `g`, `h`      |

### Using `anchor` and `anchorOffset`

While the second references the unique identifier of the first element to return, with an offset against that element, e.g.:

| Page | `anchor` | `anchorOffset` | `limit` | results       |
|------|----------|----------------|---------|---------------|
| 1    |          |                | `3`     | `a`, `b`, `c` |
| 2    | `c`      |  `1`           | `3`     | `d`, `e`, `f` |
| 3    | `f`      |  `1`           | `3`     | `g`, `h`      |

## Non-JMAP Backends

In case of multiple backends, not all will be JMAP, and they won't all support the
same pagination mechanics either.

Paginating over results that come from multiple sources is already a lot more complex
in its own right, since it requires

* performing the query over each backend
* merging the results into a single collection
* ordering them in memory
* capping the results at the requested page size (`limit`)

To illustrate this process, we have a JMAP backend that has the following elements:

|   0 |   1 |   2 |   3 |   4 |   5 |   6 |   7 |
|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `a` | `b` | `c` | `d` | `e` | `f` | `g` | `h` |

and an OpenCloud backend that has these:

|   0 |   1 |   2 |   3 |   4 |   5 |
|:---:|:---:|:---:|:---:|:---:|:---:|
| `U` | `V` | `W` | `X` | `Y` | `Z` |

When we perform a query across those:

| Page | `position` | `limit` | JMAP results | OC results | ordered | capped |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `0` | `3` | `a`, `b`, `c` | `U`, `V`, `W` | `a`, `b`, `U`, `c`, `V`, `W` | `a`, `b`, `U` |
| 2 | `3` | `3` | `d`, `e`, `f` | `X`, `Y`, `Z` | `d`, `X`, `Y`, `e`, `f`, `Z` | `d`, `X`, `Y` |

:thinking: wait a minute...

If we just naively apply this algorithm, we will skip and lose results (in this example, where have `c`, `V`, `W` gone?).

The thus cannot use the `position` and `limit` "globally" as parameters for queries for each backend.

The approach using `anchor` and `anchorOffset` seems a lot more promising, although it comes with a limitation: it's only ever possible to go to the next page, one by one, and not to jump to e.g. page 5 or page 10 at once.

| Page | `anchor` | `offset` | `limit` | JMAP results | OC results | ordered | capped |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | | | `3` | `a`, `b`, `c` | `U`, `V`, `W` | `a`, `b`, `U`, `c`, `V`, `W` | `a`, `b`, `U` |
| 2 | `b`, `U` | `1`, `1` | `3` | `c`, `d`, `e` | `V`, `W`, `X` | `c`, `V`, `W`, `d`, `e`, `X` | `c`, `V`, `W` |
| 3 | `c`, `W` | `1`, `1` | `3` | `d`, `e`, `f` | `X`, `Y`, `Z` | `d`, `e`, `X`, `Y`, `f`, `Z` | `d`, `e`, `X` |
| 4 | `e`, `X` | `1`, `1` | `3` | `f` | `Y`, `Z` | `Y`, `f`, `Z` | `Y`, `f`, `Z` |

Note that, as is apparent in the example above, we cannot simply have *one* anchor but, instead, require having *one anchor per backend* -- in this case, one for the JMAP backend, and one for the OpenCloud backend, in order to perform the next query for each of them using the correct and respective anchor.

In order to conceal that complexity as well as the fact that there are multiple backends in the first place, the best (and possibly only) approach is to use an opaque object that needs to be passed on for the "next page".

A simple and straightforward approach could be to just encode a JSON object that contains per-backend information using base64, e.g.:

```json
{
    "jmap": {
        "anchor": "b",
        "anchorOffset": 1
    },
    "oc": {
        "anchor": "U",
        "anchorOffset": 1
    }
}
```

When thrown into a base64 encoder, this results in:

```text
eyJqbWFwIjp7ImFuY2hvciI6ImIiLCJhbmNob3JPZmZzZXQiOjF9LCJvYyI6eyJhbmNob3IiOiJVIiwiYW5jaG9yT2Zmc2V0IjoxfX0K
```

A more optimized and compact object like this:

```json
{
    "jmap": "b",
    "oc": "U"
}
```

would result in a smaller base64'd opaque object:

```text
eyJqbWFwIjoiYiIsIm9jIjoiVSJ9Cg==
```

Encoding into base64 is not going to make that opaque object any smaller, to the contrary, but it will make it somewhat opaque to avoid conveying the idea that a client should be able to play around with it.

If we want to aim for compactness, a custom encoding would be most efficient, since even JSON is unnecessarily verbose as the only thing we actually want to keep track of is the `anchorOffset` for each backend.

Something like this would indeed be smaller, but not very opaque:

```text
jmap:b,oc:U
```

This approach can also handle with backends that don't all support the same pagination mechanisms: if, say, the OpenCloud backend is unable to use `anchor` and `anchorOffset` but only supports `position` and `limit` instead.

In that case, the opaque "next page" object could look like this:

```text
jmap:b,oc:2
```

And could be made opaque and URL safe using [base62 encoding](https://en.wikipedia.org/wiki/Base62):

```text
R2Wgx8cUhsw2lc
```

But in order to future-proof the format and allow more flexibility/complexity for each backend to store information that is relevant to whatever it needs to perform the pagination, JSON and base62 are the combination of choice:

```json
{"jmap":"b","oc":2}
```

Results in the following opaque object:

```text
1bt1nSpH79BZ7ZI6TK1PH1MrX
```

## API

In order to present a consistent REST API for all paginated endpoints, we should not differentiate between ones that have a single backend and those that have multiple ones, as that might also evolve over time, requiring API changes.

The common denominator is a more restricted API that allows for two operations:

* query the initial page with a given page size (limit): `/groupware/accounts/a/things?first=50`
* query the next page: `/groupware/accounts/a/things?next=...`
* query the next page: `/groupware/accounts/a/things?next=...`
* etc...

It should also be possible to query the first *zero* items in order to only calculate the total amount of items: `/groupware/accounts/a/things?first=0`

The same approach is also applicable to operations that perform queries across multiple accounts, as they pose the same issue as ones that perform queries across multiple backends.

* query the initial page: `/groupware/accounts/all/emails?first=50`
* query the next page: `/groupware/accounts/all/emails?next=...`

It is up to the endpoint to encode the "next" token appropriately, depending on the use-case:

* single account, single backend:

```json
{"p": 50, "l": 50}
```

* multiple accounts, single backend:

```json
{
    "a1": {
        "a": "naijuh7u", "o": 1, "l": 50
    },
    "a2": {
        "a": "isaicoi0", "o": 1, "l": 50
    }
}
```

* single account, multiple backends:

```json
{
    "jmap": {
        "a1": {
            "a": "naijuh7u", "o": 1, "l": 50
        }
    },
    "oc": {
        "a1": {
            "a": "oc:12291"
        }
    }
}
```

* multiple accounts, multiple backends:

```json
{
    "jmap": {
        "a1": {
            "a": "naijuh7u", "o": 1, "l": 50
        },
        "a2": {
            "a": "isaicoi0", "o": 1, "l": 50
        }
    },
    "oc": {
        "a1": {
            "a": "oc:12291"
        },
        "a2": {
            "a": "oc:8182"
        }
    }
}
```

The alternative would be to have different APIs depending on what each endpoint supports.

* endpoint with multiple backends or multiple accounts:<br>same as above with `?first=...` and `?next=...`
* endpoint with a single backend and a single account:<br>`?position=...&anchor=...&offset=...&limit=...`
