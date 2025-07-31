package opensearch

var (
	SearchHitToSearchMessageMatch = searchHitToSearchMessageMatch
	BuilderToBoolQuery            = builderToBoolQuery
)

func Convert[T any](v any) (T, error) {
	return convert[T](v)
}
