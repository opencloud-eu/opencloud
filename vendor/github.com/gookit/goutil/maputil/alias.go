package maputil

import (
	"fmt"
	"sort"
)

// Aliases implemented a simple string alias map.
//   - key: alias, value: real name
type Aliases map[string]string

// AddAlias to the Aliases map
func (as Aliases) AddAlias(alias, real string) {
	if rn, ok := as[alias]; ok {
		panic(fmt.Sprintf("The alias '%s' is already used by '%s'", alias, rn))
	}
	as[alias] = real
}

// AddAliases to the Aliases map
func (as Aliases) AddAliases(real string, aliases []string) {
	for _, a := range aliases {
		as.AddAlias(a, real)
	}
}

// AddAliasMap to the Aliases map
func (as Aliases) AddAliasMap(alias2real map[string]string) {
	for a, r := range alias2real {
		as.AddAlias(a, r)
	}
}

// HasAlias in the Aliases map
func (as Aliases) HasAlias(alias string) bool {
	if _, ok := as[alias]; ok {
		return true
	}
	return false
}

// ResolveAlias by given name.
func (as Aliases) ResolveAlias(alias string) string {
	if name, ok := as[alias]; ok {
		return name
	}
	return alias
}

// AliasesNames returns all sorted alias names.
func (as Aliases) AliasesNames() []string {
	ns := make([]string, 0, len(as))
	for alias := range as {
		ns = append(ns, alias)
	}
	sort.Strings(ns)
	return ns
}

// GroupAliases groups aliases by real name.
//
// returns: {real name -> []aliases, ...}
func (as Aliases) GroupAliases() map[string][]string {
	gaMap := make(map[string][]string)
	for alias, name := range as {
		gaMap[name] = append(gaMap[name], alias)
	}
	return gaMap
}
