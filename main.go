package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
)

func main() {
	go app()

	client()
}

var errTimeNotFound = errors.New("time not found")

type TimeStore struct {
	store sync.Map
}

func NewTimeStore() TimeStore {
	return TimeStore{
		store: sync.Map{},
	}
}

func (t *TimeStore) WriteAsString(s string) error {
	if !t.isValidUnixTime(s) {
		return errors.New("Not valid Unix time format")
	}
	t.store.Store("time", s)
	return nil
}

func (t *TimeStore) isValidUnixTime(s string) bool {
	// leading zeros?
	if len(s) > 10 || len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}

	if len(s) == 10 && s[0] >= '2' &&
		s[1] >= '1' && s[2] >= '4' && s[3] >= '7' &&
		s[4] >= '4' && s[5] >= '8' && s[6] >= '3' &&
		s[7] >= '6' && s[8] >= '4' && s[9] > '7' {
		// data in s is greater than 2,147,483,647
		return false
	}
	return true
}

func (t *TimeStore) ReadAsString() (string, error) {
	valueAsAny, ok := t.store.Load("time")
	if !ok {
		return "", fmt.Errorf("retrieve Time: %w", errTimeNotFound)
	}

	value, ok := valueAsAny.(string)
	if !ok {
		return "", errors.New("retrieve Time: type cast issue")
	}
	return value, nil
}

type TimeService struct {
	timeStore TimeStore
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

	err = t.timeStore.WriteAsString(data.String())
	if err != nil {
		http.Error(w, "Not valid Unix time", http.StatusBadRequest)
		return
	}
}

func (t *TimeService) Read(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	data, err := t.timeStore.ReadAsString()
	if err != nil {
		if errors.Is(err, errTimeNotFound) {
			http.Error(w, "Time has not been set yet.", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: time service - read: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%v", data)
}

func app() {
	timeService := TimeService{
		timeStore: NewTimeStore(),
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
	t := fmt.Sprintf("%v", time.Now().Unix())
	_, _ = http.Post(url, "text/plain", bytes.NewBufferString(t))
}

func get(url string) {
	res, _ := http.Get(url)
	var data strings.Builder
	_, _ = io.Copy(&data, res.Body)

	fmt.Println(data.String())
}
