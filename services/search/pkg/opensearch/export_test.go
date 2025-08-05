package opensearch

var (
	SearchHitToSearchMessageMatch = searchHitToSearchMessageMatch
	BuilderToBoolQuery            = builderToBoolQuery
	ExpandKQLASTNodes             = expandKQLASTNodes
)

func Convert[T any](v any) (T, error) {
	return convert[T](v)
}
