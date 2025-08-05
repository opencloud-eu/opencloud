package opensearchtest

type TableTest[G any, W any] struct {
	Name string
	Got  G
	Want W
	Err  error
	Skip bool
}
