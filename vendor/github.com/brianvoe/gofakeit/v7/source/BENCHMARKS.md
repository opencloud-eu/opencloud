go test -bench=. -benchmem \
goos: darwin \
goarch: amd64 \
pkg: github.com/brianvoe/gofakeit/v7 \
cpu: Apple M1 Max \
Table generated with tablesgenerator.com/markdown_tables File->Paste table data

| Benchmark           | Iterations| Time/Iter   | Bytes  | Allocations |
|---------------------|-----------|-------------|--------|-------------|
| BenchmarkPCG-10     | 251946703 | 4.763 ns/op | 0 B/op | 0 allocs/op |
| BenchmarkChaCha8-10 | 228052915 | 5.262 ns/op | 0 B/op | 0 allocs/op |
| BenchmarkJSF-10     | 323858558 | 3.712 ns/op | 0 B/op | 0 allocs/op |
| BenchmarkSFC-10     | 394809136 | 3.035 ns/op | 0 B/op | 0 allocs/op |
| BenchmarkOld-10     | 207714157 | 5.733 ns/op | 0 B/op | 0 allocs/op |
| BenchmarkDumb-10    | 458967214 | 2.611 ns/op | 0 B/op | 0 allocs/op |
| BenchmarkCrypto-10  | 15747936  | 77.15 ns/op | 0 B/op | 0 allocs/op |