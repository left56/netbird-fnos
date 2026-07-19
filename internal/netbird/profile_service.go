package netbird

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type ProfileRuntime struct {
	Active    bool `json:"active"`
	Connected bool `json:"connected"`
}
type ProfileDetail struct {
	Metadata              Profile         `json:"metadata"`
	Config                ProfileSettings `json:"config"`
	Runtime               ProfileRuntime  `json:"runtime"`
	Source                string          `json:"source"`
	Capabilities          Capabilities    `json:"capabilities"`
	RestartRequired       bool            `json:"restartRequired"`
	RestartRequiredFields []string        `json:"restartRequiredFields"`
}
type ProfileCreate struct {
	Name   string          `json:"name"`
	Config ProfileSettings `json:"config"`
	// ManagementURL is accepted for compatibility with the original create
	// form. New callers should use config.managementURL.
	ManagementURL      string `json:"managementURL,omitempty"`
	SetupKey           string `json:"setupKey"`
	PresharedKey       string `json:"presharedKey"`
	SelectAfterCreate  bool   `json:"selectAfterCreate"`
	ConnectAfterCreate bool   `json:"connectAfterCreate"`
}
type ProfileUpdate struct {
	Config ProfileSettings `json:"config"`
}
type ProfileCLI interface {
	Profiles(context.Context) ([]Profile, error)
	AddProfile(context.Context, string) error
	SelectProfile(context.Context, string) error
	RenameProfile(context.Context, string, string) error
	RemoveProfile(context.Context, string) error
	Connect(context.Context, ConnectOptions) error
	Disconnect(context.Context) error
}
type ProfileService struct {
	cli   ProfileCLI
	store *ProfileConfigStore
}

func NewProfileService(cli ProfileCLI, store *ProfileConfigStore) *ProfileService {
	return &ProfileService{cli, store}
}
func (s *ProfileService) List(ctx context.Context) ([]ProfileDetail, error) {
	ps, e := s.cli.Profiles(ctx)
	if e != nil {
		return []ProfileDetail{s.defaultFallback()}, nil
	}
	if len(ps) == 0 {
		return []ProfileDetail{s.defaultFallback()}, nil
	}
	out := make([]ProfileDetail, 0, len(ps))
	for _, p := range ps {
		d, e := s.detail(p)
		if e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, nil
}
func (s *ProfileService) Get(ctx context.Context, id string) (ProfileDetail, error) {
	ps, e := s.cli.Profiles(ctx)
	if e != nil {
		if id == "default" {
			return s.defaultFallback(), nil
		}
		return ProfileDetail{}, e
	}
	for _, p := range ps {
		if p.ID == id {
			return s.detail(p)
		}
	}
	if id == "default" {
		return s.defaultFallback(), nil
	}
	return ProfileDetail{}, os.ErrNotExist
}
func (s *ProfileService) defaultFallback() ProfileDetail {
	c, e := s.store.Get("default")
	if e != nil {
		c = ProfileSettings{Name: "default"}
	}
	if c.Name == "" {
		c.Name = "default"
	}
	p := Profile{ID: "default", Name: c.Name, Default: true, Active: true}
	return ProfileDetail{Metadata: p, Config: c, Runtime: ProfileRuntime{Active: true}, Source: "config-fallback", Capabilities: Capabilities{Profiles: true}}
}
func (s *ProfileService) detail(p Profile) (ProfileDetail, error) {
	c, e := s.store.Get(p.ID)
	source := "cli"
	if errors.Is(e, os.ErrNotExist) {
		// NetBird owns its profile files. The wrapper only persists the
		// whitelist UI settings it manages, so a CLI-discovered profile is
		// valid before it has ever been edited in this UI.
		c = ProfileSettings{Name: p.Name}
		source = "config-fallback"
	} else if e != nil {
		return ProfileDetail{}, e
	}
	p.Name = first(c.Name, p.Name)
	p.Default = p.Default || strings.EqualFold(p.Name, "default")
	return ProfileDetail{Metadata: p, Config: c, Runtime: ProfileRuntime{Active: p.Active, Connected: p.Connected}, Source: source, Capabilities: Capabilities{Profiles: true}}, nil
}
func (s *ProfileService) Create(ctx context.Context, in ProfileCreate) (ProfileDetail, error) {
	if !safeValue(in.Name) {
		return ProfileDetail{}, errors.New("invalid profile name")
	}
	if strings.EqualFold(in.Name, "default") {
		return ProfileDetail{}, errors.New("default profile already exists")
	}
	if in.Config.ManagementURL == "" {
		in.Config.ManagementURL = in.ManagementURL
	}
	all, e := s.cli.Profiles(ctx)
	if e != nil {
		return ProfileDetail{}, e
	}
	for _, p := range all {
		if strings.EqualFold(p.Name, in.Name) {
			return ProfileDetail{}, errors.New("profile name already exists")
		}
	}
	if e = s.cli.AddProfile(ctx, in.Name); e != nil {
		return ProfileDetail{}, e
	}
	all, e = s.cli.Profiles(ctx)
	if e != nil {
		return ProfileDetail{}, e
	}
	var p Profile
	for _, v := range all {
		if v.Name == in.Name {
			p = v
		}
	}
	if p.ID == "" {
		return ProfileDetail{}, errors.New("created profile not found")
	}
	in.Config.Name = in.Name
	if e = s.store.Put(p.ID, in.Config); e != nil {
		return ProfileDetail{}, e
	}
	if in.SetupKey != "" {
		if e = s.setSecret(p.ID, "setup-key", in.SetupKey); e != nil {
			return ProfileDetail{}, e
		}
	}
	if in.PresharedKey != "" {
		if e = s.setSecret(p.ID, "preshared-key", in.PresharedKey); e != nil {
			return ProfileDetail{}, e
		}
	}
	if in.SelectAfterCreate {
		if e = s.Select(ctx, p.ID); e != nil {
			return ProfileDetail{}, e
		}
	}
	if in.ConnectAfterCreate {
		if e = s.Connect(ctx, p.ID); e != nil {
			return ProfileDetail{}, e
		}
	}
	return s.Get(ctx, p.ID)
}
func (s *ProfileService) Update(ctx context.Context, id string, in ProfileUpdate) (map[string]any, error) {
	old, e := s.Get(ctx, id)
	if e != nil {
		return nil, e
	}
	in.Config.SetupKeyConfigured = old.Config.SetupKeyConfigured
	in.Config.PresharedKeyConfigured = old.Config.PresharedKeyConfigured
	if in.Config.Name == "" {
		in.Config.Name = old.Config.Name
	}
	if e = s.store.Put(id, in.Config); e != nil {
		return nil, e
	}
	changed := []string{}
	restart := []string{}
	if old.Config.InterfaceName != in.Config.InterfaceName {
		changed = append(changed, "interfaceName")
		restart = append(restart, "interfaceName")
	}
	if old.Config.InterfacePort != in.Config.InterfacePort {
		changed = append(changed, "interfacePort")
		restart = append(restart, "interfacePort")
	}
	if old.Config.MTU != in.Config.MTU {
		changed = append(changed, "mtu")
		restart = append(restart, "mtu")
	}
	return map[string]any{"changedFields": changed, "restartRequired": len(restart) > 0, "restartRequiredFields": restart}, nil
}
func (s *ProfileService) Select(ctx context.Context, id string) error {
	return s.cli.SelectProfile(ctx, id)
}
func (s *ProfileService) Connect(ctx context.Context, id string) error {
	d, e := s.Get(ctx, id)
	if e != nil {
		return e
	}
	if !d.Runtime.Active {
		if e = s.Select(ctx, id); e != nil {
			return e
		}
	}
	return s.cli.Connect(ctx, ConnectOptions{})
}
func (s *ProfileService) Disconnect(ctx context.Context, id string) error {
	d, e := s.Get(ctx, id)
	if e != nil {
		return e
	}
	if !d.Runtime.Active {
		return errors.New("profile is not active")
	}
	return s.cli.Disconnect(ctx)
}
func (s *ProfileService) Rename(ctx context.Context, id, name string) error {
	if !safeValue(name) {
		return errors.New("invalid profile name")
	}
	return s.cli.RenameProfile(ctx, id, name)
}
func (s *ProfileService) Delete(ctx context.Context, id string) error {
	d, e := s.Get(ctx, id)
	if e != nil {
		return e
	}
	if d.Metadata.Default || d.Runtime.Active || d.Runtime.Connected {
		return errors.New("profile cannot be deleted")
	}
	return s.cli.RemoveProfile(ctx, id)
}
func (s *ProfileService) SetSecret(id, kind, value string) error {
	if value == "" {
		return nil
	}
	return s.setSecret(id, kind, value)
}
func (s *ProfileService) setSecret(id, kind, value string) error {
	if !safeProfileID(id) || (kind != "setup-key" && kind != "preshared-key") {
		return errors.New("invalid secret")
	}
	if e := os.MkdirAll(filepath.Join(s.store.root, "secrets"), 0700); e != nil {
		return e
	}
	return os.WriteFile(filepath.Join(s.store.root, "secrets", id+"."+kind), []byte(value), 0600)
}
func (s *ProfileService) ClearSecret(id, kind string) error {
	return os.Remove(filepath.Join(s.store.root, "secrets", id+"."+kind))
}
func first(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
