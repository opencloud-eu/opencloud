package generators

import (
	"os"
	"strconv"
)

type NType int

const (
	NTYPE_BUS NType = iota
	NTYPE_KEYVALUE
	NTYPE_REGISTRY
)

func (n NType) String() string {
	return []string{"bus", "kv", "reg"}[n]
}

func GenerateConnectionName(service string, ntype NType) string {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}

	return firstNRunes(host, 5) + ":" + strconv.Itoa(os.Getpid()) + ":" + service + ":" + ntype.String()
}

func firstNRunes(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}
