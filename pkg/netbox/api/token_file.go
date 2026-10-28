package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

type tokenCache struct {
	token     string
	fetchedAt time.Time
}

// TokenFileRoundTripper fetches the auth token from a file and caches it for
// a minute, attaching it to the request when it is made
type TokenFileRoundTripper struct {
	TokenFile string
	Transport http.RoundTripper

	tokenCache *atomic.Pointer[tokenCache]
	watcher    *fsnotify.Watcher
	logger     *log.Logger
}

func NewTokenFileRoundTripper(transport http.RoundTripper, file string, logger *log.Logger) *TokenFileRoundTripper {
	tfr := &TokenFileRoundTripper{
		TokenFile:  file,
		Transport:  transport,
		tokenCache: new(atomic.Pointer[tokenCache]),
		logger:     logger,
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Warnf("unable to start file watcher on %s, continuing without: %v", file, err)
	} else {
		watcher.Add(file)
		go func() {
			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
			defer cancel()

			for {
				select {
				case <-ctx.Done():
					return
				case <-watcher.Events:
					_, err := tfr.reloadFile()
					if err != nil {
						logger.Warnf("unable to reload auth token file at %s: %v", file, err)
					} else {
						logger.Debug("reloaded auth token file based on FS event")
					}
				}
			}
		}()

		return tfr
	}

	return &TokenFileRoundTripper{
		TokenFile:  file,
		Transport:  transport,
		tokenCache: new(atomic.Pointer[tokenCache]),
		watcher:    watcher,
		logger:     logger,
	}
}

func (tfr *TokenFileRoundTripper) reloadFile() (*tokenCache, error) {
	token, err := os.ReadFile(tfr.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read auth token file: %w", err)
	}
	cacheItem := &tokenCache{
		token:     strings.TrimSpace(string(token)),
		fetchedAt: time.Now(),
	}
	if cacheItem.token == "" {
		return nil, errors.New("auth token file is empty")
	}
	tfr.tokenCache.Store(cacheItem)
	return cacheItem, nil
}

func (tfr *TokenFileRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cacheItem := tfr.tokenCache.Load()
	if cacheItem == nil || time.Since(cacheItem.fetchedAt) > time.Hour {
		var err error
		cacheItem, err = tfr.reloadFile()
		if err != nil {
			if cacheItem == nil {
				return nil, fmt.Errorf("unable to read auth token file: %w", err)
			} else {
				tfr.logger.Warnf("unable to reload auth token file at %s after cache went stale: %v", tfr.TokenFile, err)
			}
		}
	}

	req.Header.Set("Authorization", "Token "+cacheItem.token)
	return tfr.Transport.RoundTrip(req)
}
