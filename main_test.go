// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yohhoy/malloc-server/mempool"
)

func TestSetup(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.ReleaseMode)
}

func TestMalloc(t *testing.T) {
	pool = mempool.NewMemPool()
	router := router()

	mallocMemory := func(body string, resp *MallocResponse) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("POST", "/memory/malloc", bytes.NewBuffer([]byte(body)))
		r := httptest.NewRecorder()
		router.ServeHTTP(r, req)
		if resp != nil {
			err := json.Unmarshal(r.Body.Bytes(), resp)
			assert.NoError(t, err)
		}
		return r
	}

	t.Run("Malloc", func(t *testing.T) {
		var resp1 MallocResponse
		r := mallocMemory(`{"size":1024}`, &resp1)
		assert.Equal(t, 200, r.Code)
		assert.NotEqual(t, 0, resp1.Address)
		assert.Equal(t, uint64(1024), resp1.Size)

		var resp2 MallocResponse
		r = mallocMemory(`{"size":1024}`, &resp2)
		assert.Equal(t, 200, r.Code)
		assert.NotEqual(t, 0, resp2.Address)
		assert.Equal(t, uint64(1024), resp2.Size)

		assert.NotEqual(t, resp1.Address, resp2.Address)
	})
	t.Run("MallocZero", func(t *testing.T) {
		var resp MallocResponse
		r := mallocMemory(`{"size":0}`, &resp)
		assert.Equal(t, 400, r.Code)
		assert.Equal(t, uint64(0), resp.Address)
	})
	t.Run("TooLarge", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"size":%d}`, mempool.MemBlockLimit+1)

		var resp MallocResponse
		r := mallocMemory(reqBody, &resp)
		assert.Equal(t, 403, r.Code)
		assert.Equal(t, uint64(0), resp.Address)
	})
	t.Run("InvalidParam", func(t *testing.T) {
		var resp MallocResponse
		r := mallocMemory("{}", &resp)
		assert.Equal(t, 400, r.Code)
		assert.Equal(t, uint64(0), resp.Address)
	})
	t.Run("InvalidSize", func(t *testing.T) {
		var resp MallocResponse
		r := mallocMemory(`{"size":-1}`, &resp)
		assert.Equal(t, 400, r.Code)
		assert.Equal(t, uint64(0), resp.Address)
	})
}

func TestFree(t *testing.T) {
	pool = mempool.NewMemPool()
	router := router()

	freeMemory := func(body string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("POST", "/memory/free", bytes.NewBuffer([]byte(body)))
		r := httptest.NewRecorder()
		router.ServeHTTP(r, req)
		return r
	}

	t.Run("MallocFree", func(t *testing.T) {
		var respMalloc MallocResponse
		{
			reqBody := `{"size":1024}`
			req, _ := http.NewRequest("POST", "/memory/malloc", bytes.NewBuffer([]byte(reqBody)))
			r := httptest.NewRecorder()
			router.ServeHTTP(r, req)

			require.Equal(t, 200, r.Code)
			err := json.Unmarshal(r.Body.Bytes(), &respMalloc)
			require.NoError(t, err)
			require.NotEqual(t, 0, respMalloc.Address)
		}

		reqBody := fmt.Sprintf(`{"addr":%d}`, respMalloc.Address)
		r := freeMemory(reqBody)
		assert.Equal(t, 200, r.Code)
		r = freeMemory(reqBody) // double free
		assert.Equal(t, 400, r.Code)
	})
	t.Run("FreeZero", func(t *testing.T) {
		r := freeMemory(`{"addr":0}`)
		assert.Equal(t, 200, r.Code)
	})

	t.Run("InvalidAddr", func(t *testing.T) {
		r := freeMemory(`{"addr":123456}`)
		assert.Equal(t, 400, r.Code)
	})
}

func TestWriteRead(t *testing.T) {
	pool = mempool.NewMemPool()
	router := router()

	putMemory := func(url string, body string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer([]byte(body)))
		r := httptest.NewRecorder()
		router.ServeHTTP(r, req)
		return r
	}
	getMemory := func(url string, pval *byte) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("GET", url, nil)
		r := httptest.NewRecorder()
		router.ServeHTTP(r, req)
		if pval != nil {
			var resp MemReadResponse
			err := json.Unmarshal(r.Body.Bytes(), &resp)
			assert.NoError(t, err)
			*pval = resp.Value
		}
		return r
	}

	// setup
	var respMalloc MallocResponse
	{
		reqBody := `{"size":10}`
		req, _ := http.NewRequest("POST", "/memory/malloc", bytes.NewBuffer([]byte(reqBody)))
		r := httptest.NewRecorder()
		router.ServeHTTP(r, req)

		require.Equal(t, 200, r.Code)
		err := json.Unmarshal(r.Body.Bytes(), &respMalloc)
		require.NoError(t, err)
		require.NotEqual(t, 0, respMalloc.Address)
	}

	t.Run("Write", func(t *testing.T) {
		url := fmt.Sprintf("/memory/%d", respMalloc.Address)

		r := putMemory(url, `{"val":0}`)
		assert.Equal(t, 200, r.Code)
		r = putMemory(url, `{"val":255}`)
		assert.Equal(t, 200, r.Code)
		r = putMemory(url, `{"val":-1}`) // out of range
		assert.Equal(t, 400, r.Code)
		r = putMemory(url, `{"val":256}`) // out of range
		assert.Equal(t, 400, r.Code)
	})

	t.Run("WriteRead", func(t *testing.T) {
		url1 := fmt.Sprintf("/memory/%d", respMalloc.Address+4)
		url2 := fmt.Sprintf("/memory/%d", respMalloc.Address+5)

		r := putMemory(url1, `{"val":128}`)
		assert.Equal(t, 200, r.Code)
		r = putMemory(url2, `{"val":64}`)
		assert.Equal(t, 200, r.Code)

		var value byte
		r = getMemory(url1, &value)
		assert.Equal(t, 200, r.Code)
		assert.Equal(t, byte(128), value)
		r = getMemory(url2, &value)
		assert.Equal(t, 200, r.Code)
		assert.Equal(t, byte(64), value)
	})
	t.Run("Overrun", func(t *testing.T) {
		url := fmt.Sprintf("/memory/%d", respMalloc.Address+10)

		r := putMemory(url, `{"val":128}`)
		assert.Equal(t, HttpStatusUB, r.Code)
		r = getMemory(url, nil)
		assert.Equal(t, HttpStatusUB, r.Code)
	})
	t.Run("InvalidAddr", func(t *testing.T) {
		url := fmt.Sprintf("/memory/%d", respMalloc.Address-1)

		r := putMemory(url, `{"val":128}`)
		assert.Equal(t, HttpStatusUB, r.Code)
		r = getMemory(url, nil)
		assert.Equal(t, HttpStatusUB, r.Code)
	})
}
