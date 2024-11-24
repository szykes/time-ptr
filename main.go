package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-chi/chi/v5"
)

func main() {
	go app()

	client()
}

type TimePtrStore struct {
	store sync.Map
}

func NewTimePtrStore() TimePtrStore {
	return TimePtrStore{
		store: sync.Map{},
	}
}

func (t *TimePtrStore) SetPtr(ptr *time.Time) {
	t.store.Store("ptr", ptr)
}

func (t *TimePtrStore) Ptr() (*time.Time, error) {
	valueAsAny, ok := t.store.Load("ptr")
	if !ok {
		return nil, nil
	}

	value, ok := valueAsAny.(*time.Time)
	if !ok {
		return nil, errors.New("retrieve TimePtr: type cast issue")
	}
	return value, nil
}

type TimeService struct {
	timePtrStore TimePtrStore
}

func (t *TimeService) Update(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Unsupported content type, only 'text/plain' is allowed", http.StatusUnsupportedMediaType)
		return
	}

	var data strings.Builder
	_, err := io.Copy(&data, r.Body)
	if err != nil {
		log.Printf("ERROR: time service - update: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ptr, err := t.hexInStringToTimePtr(data.String())
	if err != nil {
		http.Error(w, "Not valid data, not matching with regex: ^0x((0{1})|([0-9a-fA-F]{<depends on the architecture>}))$", http.StatusBadRequest)
		return
	}
	t.timePtrStore.SetPtr(ptr)
}

func (t *TimeService) Read(w http.ResponseWriter, r *http.Request) {
	data, err := t.timePtrStore.Ptr()
	if err != nil {
		log.Printf("ERROR: time service - read: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%p", data)
}

func (t *TimeService) hexInStringToTimePtr(s string) (*time.Time, error) {
	if len(s) <= 2 || s[0] != '0' || s[1] != 'x' {
		return nil, errors.New("hex string to time ptr: not an address")
	}

	value, err := strconv.ParseUint(s[2:], 16, strconv.IntSize)
	if err != nil {
		return nil, errors.New("hex string to time ptr: not a valid address")
	}

	ptr := (*time.Time)(unsafe.Pointer(uintptr(value)))
	return ptr, nil
}

func app() {
	timeService := TimeService{
		timePtrStore: NewTimePtrStore(),
	}

	r := chi.NewRouter()

	r.Get("/time", timeService.Read)
	r.Post("/time", timeService.Update)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})

	err := http.ListenAndServe("localhost:3000", r)
	if err != nil {
		panic(err)
	}
}

func client() {
	url := "http://localhost:3000"
	urlTime := fmt.Sprintf("%s/time", url)

	waitForServer(url)

	post(urlTime)
	get(urlTime)
}

func waitForServer(url string) {
	timer := time.NewTimer(0 * time.Second)
	for {
		timer.Reset(200 * time.Millisecond)
		<-timer.C

		_, err := http.Get(url)
		if err != nil {
			var errSyscall syscall.Errno
			if errors.As(err, &errSyscall) {
				if errSyscall == syscall.ECONNREFUSED {
					continue
				}
			}
		} else {
			break
		}
	}
}

func post(url string) {
	t := time.Now()
	address := fmt.Sprintf("%p", &t)
	_, _ = http.Post(url, "text/plain", bytes.NewBufferString(address))
}

func get(url string) {
	res, _ := http.Get(url)
	var data strings.Builder
	_, _ = io.Copy(&data, res.Body)

	value, _ := strconv.ParseUint(data.String()[2:], 16, strconv.IntSize)
	ptr := (*time.Time)(unsafe.Pointer(uintptr(value)))
	fmt.Println(ptr.Unix())
}
