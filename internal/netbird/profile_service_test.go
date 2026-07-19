package netbird

import (
	"context"
	"testing"
)

type profileServiceFake struct{ profiles []Profile }

func (f profileServiceFake) Profiles(context.Context) ([]Profile, error)       { return f.profiles, nil }
func (profileServiceFake) AddProfile(context.Context, string) error            { return nil }
func (profileServiceFake) SelectProfile(context.Context, string) error         { return nil }
func (profileServiceFake) RenameProfile(context.Context, string, string) error { return nil }
func (profileServiceFake) RemoveProfile(context.Context, string) error         { return nil }
func (profileServiceFake) Connect(context.Context, ConnectOptions) error       { return nil }
func (profileServiceFake) Disconnect(context.Context) error                    { return nil }

func TestProfileListFallsBackWhenWrapperConfigIsMissing(t *testing.T) {
	svc := NewProfileService(profileServiceFake{profiles: []Profile{{ID: "8fc1e234", Name: "default", Active: true, Default: true}}}, NewProfileConfigStore(t.TempDir()))
	profiles, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].Source != "config-fallback" || !profiles[0].Metadata.Default {
		t.Fatalf("unexpected fallback profile: %#v", profiles)
	}
	if err := svc.Delete(context.Background(), "8fc1e234"); err == nil {
		t.Fatal("default profile deletion was not rejected")
	}
}
