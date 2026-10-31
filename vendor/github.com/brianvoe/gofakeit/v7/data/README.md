# Gofakeit Data

Gofakeit data set

## List

```go
List()
```

## Get/Set/Remove Data

```go
data.Get("desserts")

data.Set("desserts", map[string][]string{
    "cake":      {"chocolate", "vanilla"},
    "pie":       {"apple", "pecan"},
    "ice cream": {"strawberry", "vanilla"},
})

data.Remove("desserts")
```

## Get/Set/Remove Sub Data

```go
data.GetSubData("desserts", "cake")

data.SetSub("desserts", "cake", []string{"chocolate", "vanilla"})

data.RemoveSub("desserts", "cake")
```
