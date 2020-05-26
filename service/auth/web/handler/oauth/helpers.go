package oauth

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/store"
	"github.com/micro/micro/v2/internal/namespace"
)

const (
	joinKey           = "/"
	storePrefixState  = "state"
	storePrefixSecret = "secret"
)

// generateState generates a uuid v4 and write it to the store
func (h *Handler) generateState(ctx context.Context) (string, error) {
	code := uuid.New().String()
	key := strings.Join([]string{storePrefixState, namespace.FromContext(ctx), code}, joinKey)
	record := &store.Record{Key: key, Expiry: time.Minute * 5}
	return code, h.store.Write(record)
}

// vaidateState checks to see if a state is valid
func (h *Handler) validateState(ctx context.Context, state string) (bool, error) {
	key := strings.Join([]string{storePrefixState, namespace.FromContext(ctx), state}, joinKey)
	_, err := h.store.Read(key)
	if err == store.ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

// generateSecret for an account, this secret will be used to login the account next time the user
// logs in using oauth.
func (h *Handler) generateSecret(ctx context.Context, id string) error {
	key := strings.Join([]string{storePrefixSecret, namespace.FromContext(ctx), id}, joinKey)
	secret := []byte(uuid.New().String())
	return h.store.Write(&store.Record{Key: key, Value: secret})
}

// getSecret retrieves the auth secret for an account
func (h *Handler) getSecret(ctx context.Context, id string) (string, error) {
	key := strings.Join([]string{storePrefixSecret, namespace.FromContext(ctx), id}, joinKey)
	recs, err := h.store.Read(key)
	if err != nil {
		return "", err
	}
	return string(recs[0].Value), nil
}
