Bugfix: Don't re-escape notification email vars for each recipient

When a notification went to several recipients in one call (e.g. a group
invite to a space), each recipient past the first got subjects and bodies
with one extra layer of HTML escaping. escapeStringMap mutated its input
map, and the recipient loop reused the same map across iterations. It now
returns a new map.

https://github.com/opencloud-eu/opencloud/issues/2804
