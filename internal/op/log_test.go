package op

import (
	"context"
	"testing"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
)

func resetRelayLogStreamTokensForTest() {
	relayLogStreamTokensLock.Lock()
	relayLogStreamTokens = make(map[string]time.Time)
	relayLogStreamTokensLock.Unlock()
}

func TestRelayLogStreamTokenCreateVerifyAndRevoke(t *testing.T) {
	resetRelayLogStreamTokensForTest()

	token, err := RelayLogStreamTokenCreate()
	if err != nil {
		t.Fatalf("RelayLogStreamTokenCreate returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if !RelayLogStreamTokenVerify(token) {
		t.Fatal("expected fresh token to be valid")
	}

	RelayLogStreamTokenRevoke(token)
	if RelayLogStreamTokenVerify(token) {
		t.Fatal("expected revoked token to be invalid")
	}
}

func TestRelayLogStreamTokenConsumeIsSingleUse(t *testing.T) {
	resetRelayLogStreamTokensForTest()

	token, err := RelayLogStreamTokenCreate()
	if err != nil {
		t.Fatalf("RelayLogStreamTokenCreate returned error: %v", err)
	}

	if !RelayLogStreamTokenConsume(token) {
		t.Fatal("expected fresh token to be consumed")
	}
	if RelayLogStreamTokenConsume(token) {
		t.Fatal("expected consumed token to be rejected")
	}
	if RelayLogStreamTokenVerify(token) {
		t.Fatal("expected consumed token to be removed")
	}
}

func TestRelayLogStreamTokenVerifyRejectsExpiredToken(t *testing.T) {
	resetRelayLogStreamTokensForTest()

	token := "expired"
	relayLogStreamTokensLock.Lock()
	relayLogStreamTokens[token] = time.Now().Add(-time.Second)
	relayLogStreamTokensLock.Unlock()

	if RelayLogStreamTokenVerify(token) {
		t.Fatal("expected expired token to be invalid")
	}

	relayLogStreamTokensLock.RLock()
	_, ok := relayLogStreamTokens[token]
	relayLogStreamTokensLock.RUnlock()
	if ok {
		t.Fatal("expected expired token to be removed")
	}
}

func TestRelayLogStreamTokenConsumeRejectsExpiredToken(t *testing.T) {
	resetRelayLogStreamTokensForTest()

	token := "expired"
	relayLogStreamTokensLock.Lock()
	relayLogStreamTokens[token] = time.Now().Add(-time.Second)
	relayLogStreamTokensLock.Unlock()

	if RelayLogStreamTokenConsume(token) {
		t.Fatal("expected expired token to be rejected")
	}

	relayLogStreamTokensLock.RLock()
	_, ok := relayLogStreamTokens[token]
	relayLogStreamTokensLock.RUnlock()
	if ok {
		t.Fatal("expected expired token to be removed after consume")
	}
}

func TestRelayLogListExcludesCachedRowsAlreadyInDB(t *testing.T) {
	initOpCacheTestDB(t)
	if err := settingRefreshCache(context.Background()); err != nil {
		t.Fatalf("settingRefreshCache returned error: %v", err)
	}

	cachedLog := model.RelayLog{
		ID:               100,
		Time:             100,
		RequestModelName: "cached-and-db",
		ResponseContent:  "same-log",
	}
	olderDBLog := model.RelayLog{
		ID:               90,
		Time:             90,
		RequestModelName: "older-db",
		ResponseContent:  "older",
	}

	if err := db.GetDB().Create(&[]model.RelayLog{cachedLog, olderDBLog}).Error; err != nil {
		t.Fatalf("create relay logs returned error: %v", err)
	}

	relayLogCacheLock.Lock()
	relayLogCache = []model.RelayLog{cachedLog}
	relayLogCacheLock.Unlock()

	logs, err := RelayLogList(context.Background(), nil, nil, 1, 2)
	if err != nil {
		t.Fatalf("RelayLogList returned error: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("RelayLogList returned %d logs, want 2: %#v", len(logs), logs)
	}
	if logs[0].ID != cachedLog.ID || logs[1].ID != olderDBLog.ID {
		t.Fatalf("RelayLogList IDs = [%d,%d], want [%d,%d]", logs[0].ID, logs[1].ID, cachedLog.ID, olderDBLog.ID)
	}
}
