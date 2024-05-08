/***************************************************************
 *
 * Copyright (C) 2024, Pelican Project, Morgridge Institute for Research
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you
 * may not use this file except in compliance with the License.  You may
 * obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 ***************************************************************/

package director

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pelicanplatform/pelican/server_structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListServers(t *testing.T) {
	router := gin.Default()

	router.GET("/servers", listServers)

	serverAds.DeleteAll()
	mockOriginNamespace := mockNamespaceAds(5, "origin1")
	mockCacheNamespace := mockNamespaceAds(4, "cache1")
	serverAds.Set(mockOriginServerAd.URL.String(),
		&server_structs.Advertisement{
			ServerAd:     mockOriginServerAd,
			NamespaceAds: mockOriginNamespace,
		}, ttlcache.DefaultTTL)
	serverAds.Set(mockCacheServerAd.URL.String(),
		&server_structs.Advertisement{
			ServerAd:     mockCacheServerAd,
			NamespaceAds: mockCacheNamespace,
		}, ttlcache.DefaultTTL)

	require.True(t, serverAds.Has(mockOriginServerAd.URL.String()))
	require.True(t, serverAds.Has(mockCacheServerAd.URL.String()))

	expectedListOriginResNss := []string{}
	for _, ns := range mockOriginNamespace {
		expectedListOriginResNss = append(expectedListOriginResNss, ns.Path)
	}

	expectedListCacheResNss := []string{}
	for _, ns := range mockCacheNamespace {
		expectedListCacheResNss = append(expectedListCacheResNss, ns.Path)
	}

	expectedlistOriginRes := listServerResponse{
		Name:              mockOriginServerAd.Name,
		BrokerURL:         mockOriginServerAd.BrokerURL.String(),
		AuthURL:           mockOriginServerAd.URL.String(),
		URL:               mockOriginServerAd.URL.String(),
		WebURL:            mockOriginServerAd.WebURL.String(),
		Type:              mockOriginServerAd.Type,
		Latitude:          mockOriginServerAd.Latitude,
		Longitude:         mockOriginServerAd.Longitude,
		Writes:            mockOriginServerAd.Writes,
		DirectReads:       mockOriginServerAd.DirectReads,
		Listings:          mockOriginServerAd.Listings,
		Status:            HealthStatusUnknown,
		NamespacePrefixes: expectedListOriginResNss,
	}

	expectedlistCacheRes := listServerResponse{
		Name:              mockCacheServerAd.Name,
		BrokerURL:         mockCacheServerAd.BrokerURL.String(),
		AuthURL:           mockCacheServerAd.URL.String(),
		URL:               mockCacheServerAd.URL.String(),
		WebURL:            mockCacheServerAd.WebURL.String(),
		Type:              mockCacheServerAd.Type,
		Latitude:          mockCacheServerAd.Latitude,
		Longitude:         mockCacheServerAd.Longitude,
		Writes:            mockCacheServerAd.Writes,
		DirectReads:       mockCacheServerAd.DirectReads,
		Status:            HealthStatusUnknown,
		NamespacePrefixes: expectedListCacheResNss,
	}

	t.Run("query-origin", func(t *testing.T) {
		// Create a request to the endpoint
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/servers?server_type=origin", nil)
		router.ServeHTTP(w, req)

		// Check the response
		require.Equal(t, 200, w.Code)

		var got []listServerResponse
		err := json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		require.Equal(t, 1, len(got))
		assert.Equal(t, expectedlistOriginRes, got[0], "Response data does not match expected")
	})

	t.Run("query-cache", func(t *testing.T) {
		// Create a request to the endpoint
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/servers?server_type=cache", nil)
		router.ServeHTTP(w, req)

		// Check the response
		require.Equal(t, 200, w.Code)

		var got []listServerResponse
		err := json.Unmarshal(w.Body.Bytes(), &got)

		require.NoError(t, err)
		require.Equal(t, 1, len(got))
		assert.Equal(t, expectedlistCacheRes, got[0], "Response data does not match expected")
	})

	t.Run("query-all-with-empty-server-type", func(t *testing.T) {
		// Create a request to the endpoint
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/servers?server_type=", nil)
		router.ServeHTTP(w, req)

		// Check the response
		require.Equal(t, 200, w.Code)

		var got []listServerResponse
		err := json.Unmarshal(w.Body.Bytes(), &got)

		require.NoError(t, err)
		require.Equal(t, 2, len(got))
	})

	t.Run("query-all-without-query-param", func(t *testing.T) {
		// Create a request to the endpoint
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/servers", nil)
		router.ServeHTTP(w, req)

		// Check the response
		require.Equal(t, 200, w.Code)

		var got []listServerResponse
		err := json.Unmarshal(w.Body.Bytes(), &got)

		require.NoError(t, err)
		require.Equal(t, 2, len(got))
	})

	t.Run("query-with-invalid-param", func(t *testing.T) {
		// Create a request to the endpoint
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/servers?server_type=staging", nil)
		router.ServeHTTP(w, req)

		// Check the response
		require.Equal(t, 400, w.Code)
	})
}

func TestHandleDisableServerToggle(t *testing.T) {
	cleanupMap := func() {
		disabledServersMutex.Lock()
		defer disabledServersMutex.Unlock()
		disabledServers = map[string]disabledReason{}
	}
	t.Cleanup(cleanupMap)
	router := gin.Default()
	router.PATCH("/servers", handleDisableServerToggle)

	t.Run("disable-server-success", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()
		mockServerUrl := "https://mock-origin.org:8444"
		reqBody := patchServerRequest{Disabled: true}
		reqBodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)
		req, _ := http.NewRequest("PATCH", "/servers?serverUrl="+url.QueryEscape(mockServerUrl), bytes.NewReader(reqBodyBytes))

		router.ServeHTTP(w, req)

		require.Equal(t, 200, w.Code)

		disabledServersMutex.RLock()
		defer disabledServersMutex.RUnlock()
		assert.Equal(t, tempDisabled, disabledServers["https://mock-origin.org:8444"])
	})
	t.Run("disable-server-w-permDisabled-returns-400", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()

		mockServerUrl := "https://mock-perm-disabled.org:8444"

		disabledServersMutex.Lock()
		disabledServers[mockServerUrl] = permDisabeld
		disabledServersMutex.Unlock()

		reqBody := patchServerRequest{Disabled: true}
		reqBodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)
		req, _ := http.NewRequest("PATCH", "/servers?serverUrl="+url.QueryEscape(mockServerUrl), bytes.NewReader(reqBodyBytes))
		router.ServeHTTP(w, req)

		require.Equal(t, 400, w.Code)

		disabledServersMutex.RLock()
		defer disabledServersMutex.RUnlock()
		assert.Equal(t, permDisabeld, disabledServers[mockServerUrl])

		resB, err := io.ReadAll(w.Body)
		require.NoError(t, err)
		assert.Contains(t, string(resB), "Can't disable a server that already has been disabled")
	})
	t.Run("disable-server-w-tempDisabled-returns-400", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()

		mockServerUrl := "https://mock-temp-disabled.org:8444"

		disabledServersMutex.Lock()
		disabledServers[mockServerUrl] = tempDisabled
		disabledServersMutex.Unlock()

		reqBody := patchServerRequest{Disabled: true}
		reqBodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)
		req, _ := http.NewRequest("PATCH", "/servers?serverUrl="+url.QueryEscape(mockServerUrl), bytes.NewReader(reqBodyBytes))
		router.ServeHTTP(w, req)

		require.Equal(t, 400, w.Code)

		disabledServersMutex.RLock()
		defer disabledServersMutex.RUnlock()
		assert.Equal(t, tempDisabled, disabledServers[mockServerUrl])

		resB, err := io.ReadAll(w.Body)
		require.NoError(t, err)
		assert.Contains(t, string(resB), "Can't disable a server that already has been disabled")
	})
	t.Run("disable-tempEnabled-server-success", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()

		mockServerUrl := "https://mock-temp-allowed.org:8444"

		disabledServersMutex.Lock()
		disabledServers[mockServerUrl] = tempEnabled
		disabledServersMutex.Unlock()

		reqBody := patchServerRequest{Disabled: true}
		reqBodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("PATCH", "/servers?serverUrl="+url.QueryEscape(mockServerUrl), bytes.NewReader(reqBodyBytes))
		router.ServeHTTP(w, req)

		require.Equal(t, 200, w.Code)

		disabledServersMutex.RLock()
		defer disabledServersMutex.RUnlock()
		assert.Equal(t, permDisabeld, disabledServers[mockServerUrl])
	})
	t.Run("disable-without-serverUrl-returns-400", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/servers", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, 400, w.Code)
		resB, err := io.ReadAll(w.Body)
		require.NoError(t, err)
		assert.Contains(t, string(resB), "'serverUrl' is a required query parameter")
	})

	t.Run("disable-without-body-returns-400", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()
		mockServerUrl := "https://mock-origin.org:8444"
		req, _ := http.NewRequest("PATCH", "/servers?serverUrl="+url.QueryEscape(mockServerUrl), nil)
		router.ServeHTTP(w, req)

		require.Equal(t, 400, w.Code)
		resB, err := io.ReadAll(w.Body)
		require.NoError(t, err)
		assert.Contains(t, string(resB), "Failed to bind reqeust body")
	})

	/****************************
	 * Enable a disabled server *
	 ****************************/
	t.Run("enable-server-that-dne", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()
		mockServerUrl := "https://mock-origin.org:8444"
		reqBody := patchServerRequest{Disabled: false}
		reqBodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)
		req, _ := http.NewRequest("PATCH", "/servers?serverUrl="+url.QueryEscape(mockServerUrl), bytes.NewReader(reqBodyBytes))

		router.ServeHTTP(w, req)

		require.Equal(t, 400, w.Code)
		resB, err := io.ReadAll(w.Body)
		require.NoError(t, err)
		assert.Contains(t, string(resB), "Can't enable a server that is not disabled or does not exist")
	})
	t.Run("enable-server-w-permDisabled", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()
		mockServerUrl := "https://mock-perm-disabled.org:8444"

		reqBody := patchServerRequest{Disabled: false}
		reqBodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("PATCH", "/servers?serverUrl="+url.QueryEscape(mockServerUrl), bytes.NewReader(reqBodyBytes))
		disabledServersMutex.Lock()
		disabledServers[mockServerUrl] = permDisabeld
		disabledServersMutex.Unlock()
		router.ServeHTTP(w, req)

		require.Equal(t, 200, w.Code)

		disabledServersMutex.RLock()
		defer disabledServersMutex.RUnlock()
		assert.Equal(t, tempEnabled, disabledServers[mockServerUrl])
	})
	t.Run("enable-server-w-tempDisabled", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()
		mockServerUrl := "https://mock-temp-disabled.org:8444"
		reqBody := patchServerRequest{Disabled: false}
		reqBodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("PATCH", "/servers?serverUrl="+url.QueryEscape(mockServerUrl), bytes.NewReader(reqBodyBytes))
		disabledServersMutex.Lock()
		disabledServers[mockServerUrl] = tempDisabled
		disabledServersMutex.Unlock()
		router.ServeHTTP(w, req)

		require.Equal(t, 200, w.Code)

		disabledServersMutex.RLock()
		defer disabledServersMutex.RUnlock()
		assert.Empty(t, disabledServers[mockServerUrl])
	})
	t.Run("enable-tempEnabled-server-400", func(t *testing.T) {
		defer cleanupMap()
		w := httptest.NewRecorder()

		mockServerUrl := "https://mock-temp-disabled.org:8444"
		reqBody := patchServerRequest{Disabled: false}
		reqBodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest("PATCH", "/servers?serverUrl="+url.QueryEscape(mockServerUrl), bytes.NewReader(reqBodyBytes))
		disabledServersMutex.Lock()
		disabledServers[mockServerUrl] = tempEnabled
		disabledServersMutex.Unlock()
		router.ServeHTTP(w, req)

		require.Equal(t, 400, w.Code)

		disabledServersMutex.RLock()
		defer disabledServersMutex.RUnlock()
		assert.Equal(t, tempEnabled, disabledServers[mockServerUrl])

		resB, err := io.ReadAll(w.Body)
		require.NoError(t, err)
		assert.Contains(t, string(resB), "Can't enable a server that already has been enabled")
	})
}
