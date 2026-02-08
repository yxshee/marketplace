package auth

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/platform/identifier"
)

var (
	ErrEmailInUse          = errors.New("email already in use")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserNotFound        = errors.New("user not found")
	ErrVendorAlreadyLinked = errors.New("vendor already linked")
)

// User is the in-memory auth aggregate root used during foundation phase.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         Role
	VendorID     *string
	CreatedAt    time.Time
}

// Session is the auth refresh-session state tracked server-side.
type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	ExpiresAt        time.Time
}

// Service provides first-party auth and session management behavior.
type Service struct {
	mu             sync.RWMutex
	usersByID      map[string]User
	usersByEmail   map[string]string
	sessionsByID   map[string]Session
	bootstrapRoles map[string]Role
}

func NewService(bootstrapRoles map[string]Role) *Service {
	normalized := make(map[string]Role, len(bootstrapRoles))
	for email, role := range bootstrapRoles {
		normalized[strings.ToLower(strings.TrimSpace(email))] = role
	}

	return &Service{
		usersByID:      make(map[string]User),
		usersByEmail:   make(map[string]string),
		sessionsByID:   make(map[string]Session),
		bootstrapRoles: normalized,
	}
}

func BuildBootstrapRoleMap(superAdmins, support, finance, catalogModerators string) map[string]Role {
	assignments := make(map[string]Role)
	assign(assignments, superAdmins, RoleSuperAdmin)
	assign(assignments, support, RoleSupport)
	assign(assignments, finance, RoleFinance)
	assign(assignments, catalogModerators, RoleCatalogModerator)
	return assignments
}

func assign(assignments map[string]Role, emails string, role Role) {
	for _, raw := range strings.Split(emails, ",") {
		email := strings.ToLower(strings.TrimSpace(raw))
		if email == "" {
			continue
		}
		assignments[email] = role
	}
}

func (s *Service) Register(email, plainPassword string) (User, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return User{}, ErrInvalidCredentials
	}

	hash, err := HashPassword(plainPassword)
	if err != nil {
		return User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.usersByEmail[normalized]; exists {
		return User{}, ErrEmailInUse
	}

	role := RoleBuyer
	if bootstrappedRole, exists := s.bootstrapRoles[normalized]; exists {
		role = bootstrappedRole
	}

	user := User{
		ID:           identifier.New("usr"),
		Email:        normalized,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
	}

	s.usersByID[user.ID] = user
	s.usersByEmail[normalized] = user.ID
	return user, nil
}

func (s *Service) Authenticate(email, plainPassword string) (User, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))

	s.mu.RLock()
	userID, exists := s.usersByEmail[normalized]
	if !exists {
		s.mu.RUnlock()
		return User{}, ErrInvalidCredentials
	}
	user := s.usersByID[userID]
	s.mu.RUnlock()

	if !VerifyPassword(user.PasswordHash, plainPassword) {
		return User{}, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Service) GetUserByID(userID string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.usersByID[userID]
	return user, exists
}

func (s *Service) AttachVendor(userID, vendorID string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.usersByID[userID]
	if !exists {
		return User{}, ErrUserNotFound
	}

	if user.VendorID != nil && *user.VendorID != vendorID {
		return User{}, ErrVendorAlreadyLinked
	}

	user.Role = RoleVendorOwner
	userVendorID := vendorID
	user.VendorID = &userVendorID
	s.usersByID[userID] = user
	return user, nil
}

func (s *Service) SaveSession(session Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionsByID[session.ID] = session
}

func (s *Service) GetSession(sessionID string) (Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessionsByID[sessionID]
	return session, exists
}

func (s *Service) DeleteSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessionsByID, sessionID)
}
