# time-ptr

I don't recommend this solutions due to multiple reasons:
* A uintptr is an integer, not a reference. Even if a uintptr holds the address of some object, the garbage collector will not update that uintptr's value if the object moves, nor will that uintptr keep the object from being reclaimed. From: https://pkg.go.dev/unsafe#Pointer I think this requires extra care with using `runtime.KeepAlive()` on the object.
* The address is a virtual address, so this has meaning only within the virtual memory map of the specific process. If other process reaches the same physical memory via paging, the other process will have a different virtual address for that data.
* Sharing address of a memory is an efficient way of memory handling and it can work on low-level communication like using library, calling OS API functions, etc. well but on high-level communication like all protocols they use TCP/UDP, etc. I think this is an anti-pattern.

## Run

I have delivered two solutions:
* `main.go-map` is identical to `main.go` - this uses `sync.Map` in `TimePtrStore`
* `main.go-chan` - this uses `chan`s in `TimePtrStore`

> The two solutions differ only in the implementation of `TimePtrStore`.

```
go run .
```

## Test

```
go test .
```

```
go test -race -gcflags=-d=checkptr=0 -run TestTime_Concurrency .
```
