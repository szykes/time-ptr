package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit Tests

func TestTimeStore(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name    string
		value   string
		isValid bool
	}{
		{
			name: "check empty string",
		},
		{
			name:    "set valid min time",
			value:   "0",
			isValid: true,
		},
		{
			name:    "set just a time",
			value:   "1732483484",
			isValid: true,
		},
		{
			name:    "set valid max time",
			value:   "2147483647",
			isValid: true,
		},
		{
			name:  "set int max + 1",
			value: "2147483648",
		},
		{
			name:  "set longer",
			value: "21474836470",
		},
		// more TCs with letter(s), -1, leading zeros, etc.
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := NewTimeStore()

			s, err := store.ReadAsString()
			assert.Empty(t, s, tc.name)
			assert.ErrorIs(t, err, errTimeNotFound, tc.name)

			err = store.WriteAsString(tc.value)
			if tc.isValid {
				assert.NoError(t, err, tc.name)
			} else {
				assert.Error(t, err, tc.name)
			}

			s, err = store.ReadAsString()
			if tc.isValid {
				assert.Equal(t, tc.value, s, tc.name)
				assert.NoError(t, err, tc.name)
			} else {
				assert.Empty(t, s, tc.name)
				assert.ErrorIs(t, err, errTimeNotFound, tc.name)
			}
		})
	}
}

// Integration Tests

func TestTime(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name         string
		value        string
		contentTypes []string
		getCode      int
		postCode     int
	}{
		{
			name:    "only GET init value",
			getCode: http.StatusNotFound,
		},
		{
			name:     "use valid time",
			value:    "1732483484",
			getCode:  http.StatusOK,
			postCode: http.StatusOK,
		},
		{
			name:     "use invalid time",
			value:    "21474836470",
			postCode: http.StatusBadRequest,
			getCode:  http.StatusNotFound,
		},
		{
			name:         "check content types",
			value:        "1732483484",
			postCode:     http.StatusUnsupportedMediaType,
			contentTypes: []string{"text/html", "text/javascript", "application/x-www-form-urlencoded"},
		},
	}

	get := func(t *testing.T, timeService *TimeService, code int, value, tcName string) {
		request, _ := http.NewRequest("GET", "/time", nil)
		response := httptest.NewRecorder()
		timeService.Read(response, request)
		assert.Equal(t, code, response.Code, tcName)
		if code == http.StatusOK {
			assert.Equal(t, value, response.Body.String(), tcName)
		} else {
			assert.Equal(t, "Time has not been set yet.\n", response.Body.String(), tcName)
		}
		assert.Contains(t, response.Header().Get("Content-Type"), "text/plain", tcName)
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
			t.Parallel()

			timeService := TimeService{
				timeStore: NewTimeStore(),
			}

			if len(tc.contentTypes) > 0 {
				for _, contentType := range tc.contentTypes {
					post(t, &timeService, tc.postCode, tc.value, contentType, tc.name+" - test Content-Type")
				}
				return
			}

			if tc.postCode == 0 {
				get(t, &timeService, tc.getCode, tc.value, tc.name+" - only get")
				return
			}

			post(t, &timeService, tc.postCode, tc.value, "text/plain", tc.name+" - 1st post")
			get(t, &timeService, tc.getCode, tc.value, tc.name+" - 1st get")
			post(t, &timeService, tc.postCode, tc.value, "text/plain", tc.name+" - 2nd post")
			post(t, &timeService, tc.postCode, tc.value, "text/plain", tc.name+" - 3rd post")
			get(t, &timeService, tc.getCode, tc.value, tc.name+" - 2nd get")
			get(t, &timeService, tc.getCode, tc.value, tc.name+" - 3rd get")
		})
	}
}

func TestTime_Concurrency(t *testing.T) {
	t.Parallel()

	get := func(timeService *TimeService, wg *sync.WaitGroup) {
		request, _ := http.NewRequest("GET", "/time", nil)
		response := httptest.NewRecorder()
		timeService.Read(response, request)
		wg.Done()
	}
	post := func(timeService *TimeService, wg *sync.WaitGroup) {
		request, _ := http.NewRequest("POST", "/time", bytes.NewBufferString("1732483484"))
		request.Header.Set("Content-Type", "text/plain")
		response := httptest.NewRecorder()
		timeService.Update(response, request)
		wg.Done()
	}

	timeService := TimeService{
		timeStore: NewTimeStore(),
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go get(&timeService, &wg)
		go post(&timeService, &wg)
	}
	wg.Wait()
}
