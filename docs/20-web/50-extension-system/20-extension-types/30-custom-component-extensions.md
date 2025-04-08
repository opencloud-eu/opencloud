---
title: 'Custom component extensions'
sidebar_position: 3
id: custom-component-extensions
---



## Extension Type CustomComponent

CustomComponent extensions need to define one or multiple `extensionPointId`s as render target. A `CustomComponentTarget` component for this very
extension point needs to be mounted in the current view.

### Configuration

To define a custom component extension, you implement the `CustomComponentExtension` interface.
Here's what it looks like:

```typescript
interface CustomComponentExtension {
  id: string
  type: 'customComponent'
  extensionPointIds?: string[]
  content: Slot | Component
}
```

For `id`, `type`, and `extensionPointIds`, please see [extension base section](./../index.md) in the top level docs.

The `content` property specifies a render function or a Component for the target extension point.

### Example

A simple example for a custom component extension could be a `NyanCat` progress bar component, being
targeted at the `global-progress-bar` extension point as render target.

```typescript
const extension = {
  id: 'com.github.opencloud-eu.web.app.progress-bars.nyan-cat',
  type: 'customComponent',
  extensionPointIds: ['app.runtime.global-progress-bar'],
  content: (slots) => [h(NyanCat, slots)],
  userPreference: {
    optionLabel: $gettext('Nyan Cat progress bar')
  }
}
```

The `content` property in this example can also be defined as `content: NyanCat`.
