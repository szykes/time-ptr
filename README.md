# time-ptr

I don't recommend this solution due to multiple reasons:
* The Unix time is UTC without having information about TZ
* The date cannot be eariler than 01/01/1970, and later than 03:14:08 UTC on 19 January 2038
* (As string data this is not clear to the reader what data this is without adding more info. Meanwhile, the "2024-11-25 20:43:44 +0100 CET" is straightforward.)

## Run

I have delivered two solutions:
* `main.go-map` is identical to `main.go` - this uses `sync.Map` in `TimePtrStore`
* `main.go-chan` - this uses `chan`s in `TimePtrStore`

```
go run .
```

## Test

```
go test -race .
```
