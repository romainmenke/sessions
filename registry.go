package sessions

import (
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
)

// Registry -------------------------------------------------------------------

// sessionInfo stores a session tracked by the registry.
type sessionInfo struct {
	s *Session
	e error
}

// contextKey is the type used to store the registry in the context.
type registryKeyType int

// registryKey is the key used to store the registry in the context.
const registryKey registryKeyType = 0

func ContextWithRegistry(ctx context.Context, r *http.Request) context.Context {
	registry := RegistryFromContext(ctx)
	if registry != nil {
		return ctx
	}
	newRegistry := &Registry{
		request:  r,
		sessions: make(map[string]sessionInfo),
	}
	return context.WithValue(ctx, registryKey, newRegistry)
}

// GetRegistry returns a registry instance for the current request.
func RegistryFromContext(ctx context.Context) *Registry {
	v := ctx.Value(registryKey)
	if v == nil {
		return nil
	}

	if registry, ok := v.(*Registry); ok {
		return registry
	}

	return nil
}

// Helpers --------------------------------------------------------------------

func init() {
	gob.Register([]interface{}{})
}

// Save saves all sessions used during the current request.
func Save(r *http.Request, w http.ResponseWriter) error {
	return RegistryFromContext(r.Context()).Save(w)
}

// Registry stores sessions used during a request.
type Registry struct {
	request  *http.Request
	sessions map[string]sessionInfo
}

// Get registers and returns a session for the given name and session store.
//
// It returns a new session if there are no sessions registered for the name.
func (s *Registry) Get(store Store, name string) (session *Session, err error) {
	if !isCookieNameValid(name) {
		return nil, fmt.Errorf("sessions: invalid character in cookie name: %s", name)
	}
	if info, ok := s.sessions[name]; ok {
		session, err = info.s, info.e
	} else {
		session, err = store.New(s.request, name)
		session.name = name
		s.sessions[name] = sessionInfo{s: session, e: err}
	}
	session.store = store
	return
}

// Save saves all sessions registered for the current request.
func (s *Registry) Save(w http.ResponseWriter) error {
	var errMulti MultiError
	for name, info := range s.sessions {
		session := info.s
		if session.store == nil {
			errMulti = append(errMulti, fmt.Errorf(
				"sessions: missing store for session %q", name))
		} else if err := session.store.Save(s.request, w, session); err != nil {
			errMulti = append(errMulti, fmt.Errorf(
				"sessions: error saving session %q -- %v", name, err))
		}
	}
	if errMulti != nil {
		return errMulti
	}
	return nil
}
