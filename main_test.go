package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Unit Tests

func TestTimePtrStore(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name   string
		values []*time.Time
	}{
		{
			name: "check default value",
		},
		{
			name:   "set one non-null ptr",
			values: []*time.Time{&time.Time{}},
		},
		{
			name:   "set one null ptr",
			values: []*time.Time{nil},
		},
		{
			name:   "set multiple non-null ptrs",
			values: []*time.Time{&time.Time{}, &time.Time{}, &time.Time{}},
		},
		// Why is the 'set different type in sync.Map' case missing to reach the 100% coverage?
		// I think the purpose of unit test is to test the unit via its interfaces (aka black box) and not the internal codes directly (white box).
		// If the unit test is a white box testing, it prohibits any meaningful refactors due to the test.
		// BTW: I think my two solutions demonstrate well why the black box is a good approach because
		// the same test function can test the solutions without any change.
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := NewTimePtrStore()

			if len(tc.values) == 0 {
				ptr, err := store.Ptr()
				assert.Nil(t, ptr, tc.name)
				assert.NoError(t, err, tc.name)
				return
			}

			for _, value := range tc.values {
				store.SetPtr(value)

				ptr, err := store.Ptr()
				assert.Equal(t, value, ptr, tc.name)
				assert.NoError(t, err, tc.name)
			}
		})
	}
}

// Integration Tests

func TestTime(t *testing.T) {
	tcs := []struct {
		name         string
		values       []string
		contentTypes []string
		getCode      int
		postCode     int
	}{
		{
			name:    "only GET init value",
			values:  []string{"0x0"},
			getCode: http.StatusOK,
		},
		{
			name:     "use one digits only value",
			values:   []string{"0x1234567890"},
			getCode:  http.StatusOK,
			postCode: http.StatusOK,
		},
		{
			name:     "use one lower case letters only value",
			values:   []string{"0xabcedfabcd"},
			getCode:  http.StatusOK,
			postCode: http.StatusOK,
		},
		// TODO: continue the pattern
		{
			name:         "check content types",
			values:       []string{"0x1234567890"},
			postCode:     http.StatusUnsupportedMediaType,
			contentTypes: []string{"text/html", "text/javascript", "application/x-www-form-urlencoded"},
		},
	}

	get := func(t *testing.T, timeService *TimeService, code int, value, tcName string) {
		request, _ := http.NewRequest("GET", "/time", nil)
		response := httptest.NewRecorder()
		timeService.Read(response, request)
		assert.Equal(t, code, response.Code, tcName)
		assert.Equal(t, value, response.Body.String(), tcName)
		assert.Equal(t, "text/plain", response.Header().Get("Content-Type"), tcName)
	}
	post := func(t *testing.T, timeService *TimeService, code int, value, contentType, tcName string) {
		request, _ := http.NewRequest("POST", "/time", bytes.NewBufferString(value))
		request.Header.Set("Content-Type", contentType)
		response := httptest.NewRecorder()
		timeService.Update(response, request)
		assert.Equal(t, code, response.Code, tcName)
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			timeService := TimeService{
				timePtrStore: NewTimePtrStore(),
			}

			if len(tc.contentTypes) > 0 {
				for _, contentType := range tc.contentTypes {
					post(t, &timeService, tc.postCode, tc.values[0], contentType, tc.name+" - test Content-Type")
				}
				return
			}

			for _, value := range tc.values {
				if tc.postCode == 0 {
					get(t, &timeService, tc.getCode, value, tc.name+" - only get")
					return
				}

				post(t, &timeService, tc.postCode, value, "text/plain", tc.name+" - 1st post")
				get(t, &timeService, tc.getCode, value, tc.name+" - 1st get")
				post(t, &timeService, tc.postCode, value, "text/plain", tc.name+" - 2nd post")
				post(t, &timeService, tc.postCode, value, "text/plain", tc.name+" - 3rd post")
				get(t, &timeService, tc.getCode, value, tc.name+" - 2nd get")
				get(t, &timeService, tc.getCode, value, tc.name+" - 3rd get")
			}
		})
	}
}

func TestTime_Concurrency(t *testing.T) {
	get := func(timeService *TimeService, wg *sync.WaitGroup) {
		request, _ := http.NewRequest("GET", "/time", nil)
		response := httptest.NewRecorder()
		timeService.Read(response, request)
		wg.Done()
	}
	post := func(timeService *TimeService, wg *sync.WaitGroup) {
		t := time.Now()
		value := fmt.Sprintf("%p", &t)
		request, _ := http.NewRequest("POST", "/time", bytes.NewBufferString(value))
		request.Header.Set("Content-Type", "text/plain")
		response := httptest.NewRecorder()
		timeService.Update(response, request)
		wg.Done()
	}

	timeService := TimeService{
		timePtrStore: NewTimePtrStore(),
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go get(&timeService, &wg)
		go post(&timeService, &wg)
	}
	wg.Wait()
}
