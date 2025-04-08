---
title: 'Search extensions'
sidebar_position: 7
id: search-extensions
---



## Search extensions

One possible extension type is search. Registered search extensions are available when using the search field in the topbar. A search extension can consist of a
`list` and a `preview` search. The result of a `preview` search is shown below the search input field while the `list` search result is

### Configuration

An example of a search extension configuration can be found below:

```typescript
interface SearchExtension {
  id: string
  type: 'search'
  extensionPointIds?: string[]
  searchProvider: {
    id: string
    available: boolean
    displayName?: string
    previewSearch?: SearchPreview // See SearchPreview section below
    listSearch?: SearchList // See SearchList section below
  }
}
```

For `id`, `type`, and `extensionPointIds`, please see [extension base section](./../index.md) in the top level docs.

The `searchProvider` object configures the actual provider. It consist of the following:

- `id` - Since your extension has an `id` and can only have one searchProvider, you can reuse the same value
- `available` - Can be used to programmatically disable/enable any searchProvider, e.g. by dynamically checking backend capabilities
- `displayName` - Optional, used to add a small hint to indicate the connection between search providers and their corresponding results
- `previewSearch` - See below
- `listSearch` - See below

#### ListSearch

The listSearch object consists of:

- `component` - Vue component that can render the values from the SearchResult below
- `search(term: string)` - Function that exectues the search, based on a given term. The term is formatted in [KQL](https://docs.opencloud.eu/services/search/#query-language). Please note that the returned values needs to be formatted to fit either `SearchResource` or `GenericSearchResultItem` type

#### PreviewSearch

The previewSearch object extends the listSearch with one additional attribute:

- `available` - Indicates whether a preview underneath the search bar is available for this search provider

## Example

The following example shows how a search extension that queries a Solr search engine could look like. Note that the extension is wrapped inside a Vue composable so it can easily be reused. All helper types and composables are being provided via the [web-pkg](https://github.com/opencloud-eu/web/tree/main/packages/web-pkg) and the [web-client](https://github.com/opencloud-eu/web/tree/main/packages/web-client) packages.

```typescript
export const useSolrSearchExtension = () => {
  const { $gettext } = useGettext()

  const previewSearch: SearchPreview = {
    component: SolarSearchComponent, // see the next example snippet
    available: () => true,
    search: (term) => {
      // actual search implementation
      console.log('Querying solr search engine...')
    }
  }

  const searchProvider: SearchProvider = {
    id: 'solr-search',
    available: true,
    displayName: 'Solr Search',
    previewSearch
  }

  const extension = computed<SearchExtension>(() => ({
    id: 'com.github.opencloud-eu.web.solr-search',
    type: 'search',
    searchProvider
  }))

  return { extension }
}
```

The search component for the preview search container may look like this:

```Vue
<template>
  <resource-list-item :resource="resource" />
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from 'vue'
import { Resource } from '@opencloud-eu/web-client'
import { SearchResultValue, ResourceListItem } from '@opencloud-eu/web-pkg'

export default defineComponent({
  name: 'SolarSearchComponent',
  components: { ResourceListItem },
  props: {
    searchResult: {
      type: Object as PropType<SearchResultValue>,
      default: () => ({})
    }
  },
  setup(props) {
    const resource = computed<Resource>(() => props.searchResult.data)

    return { resource }
  }
})
</script>

```

The extension can then be registered in any app like so:

```typescript
export default defineWebApplication({
  setup() {
    const { extension } = useSolrSearchExtension()

    return {
      appInfo: {
        name: $gettext('Solr search app'),
        id: 'solrs-search-app'
      },
      extensions: computed(() => [unref(extension)])
    }
  }
})
```
