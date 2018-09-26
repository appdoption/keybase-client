// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package engine

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/keybase/client/go/libkb"
	keybase1 "github.com/keybase/client/go/protocol/keybase1"
	"github.com/stretchr/testify/require"
)

func runIdentify(tc *libkb.TestContext, username string) (idUI *FakeIdentifyUI, res *keybase1.Identify2ResUPK2, err error) {
	idUI = &FakeIdentifyUI{}
	arg := keybase1.Identify2Arg{
		UserAssertion:    username,
		AlwaysBlock:      true,
		IdentifyBehavior: keybase1.TLFIdentifyBehavior_CLI,
	}

	uis := libkb.UIs{
		LogUI:      tc.G.UI.GetLogUI(),
		IdentifyUI: idUI,
	}

	eng := NewResolveThenIdentify2(tc.G, &arg)
	m := NewMetaContextForTest(*tc).WithUIs(uis)
	err = RunEngine2(m, eng)
	if err != nil {
		return idUI, nil, err
	}
	res, err = eng.Result()
	if err != nil {
		return idUI, nil, err
	}
	return idUI, res, nil
}

func checkAliceProofs(tb libkb.TestingTB, idUI *FakeIdentifyUI, user *keybase1.UserPlusKeysV2) {
	checkKeyedProfile(tb, idUI, user, "alice", map[string]string{
		"github":  "kbtester2",
		"twitter": "tacovontaco",
	})
}

func checkBobProofs(tb libkb.TestingTB, idUI *FakeIdentifyUI, user *keybase1.UserPlusKeysV2) {
	checkKeyedProfile(tb, idUI, user, "bob", map[string]string{
		"github":  "kbtester1",
		"twitter": "kbtester1",
	})
}

func checkCharlieProofs(tb libkb.TestingTB, idUI *FakeIdentifyUI, user *keybase1.UserPlusKeysV2) {
	checkKeyedProfile(tb, idUI, user, "charlie", map[string]string{
		"github":  "tacoplusplus",
		"twitter": "tacovontaco",
	})
}

func checkDougProofs(tb libkb.TestingTB, idUI *FakeIdentifyUI, user *keybase1.UserPlusKeysV2) {
	checkKeyedProfile(tb, idUI, user, "doug", nil)
}

func checkKeyedProfile(tb libkb.TestingTB, idUI *FakeIdentifyUI, them *keybase1.UserPlusKeysV2, name string, expectedProofs map[string]string) {
	if them == nil {
		tb.Fatal("nil 'them' user")
	}
	exported := &keybase1.User{
		Uid:      them.GetUID(),
		Username: them.GetName(),
	}
	if !reflect.DeepEqual(idUI.User, exported) {
		tb.Fatal("LaunchNetworkChecks User not equal to result user.", idUI.User, exported)
	}

	if !reflect.DeepEqual(expectedProofs, idUI.Proofs) {
		tb.Fatal("Wrong proofs.", expectedProofs, idUI.Proofs)
	}
}

func checkDisplayKeys(t *testing.T, idUI *FakeIdentifyUI, callCount, keyCount int) {
	if idUI.DisplayKeyCalls != callCount {
		t.Errorf("DisplayKey calls: %d.  expected %d.", idUI.DisplayKeyCalls, callCount)
	}

	if len(idUI.Keys) != keyCount {
		t.Errorf("keys: %d, expected %d.", len(idUI.Keys), keyCount)
		for k, v := range idUI.Keys {
			t.Logf("key: %+v, %+v", k, v)
		}
	}
}

func TestIdAlice(t *testing.T) {
	tc := SetupEngineTest(t, "id")
	defer tc.Cleanup()
	idUI, result, err := runIdentify(&tc, "t_alice")
	require.NoError(t, err)
	checkAliceProofs(t, idUI, &result.Upk.Current)
	checkDisplayKeys(t, idUI, 1, 1)
}

func TestIdBob(t *testing.T) {
	tc := SetupEngineTest(t, "id")
	defer tc.Cleanup()
	idUI, result, err := runIdentify(&tc, "t_bob")
	require.NoError(t, err)
	checkBobProofs(t, idUI, &result.Upk.Current)
	checkDisplayKeys(t, idUI, 1, 1)
}

func TestIdCharlie(t *testing.T) {
	tc := SetupEngineTest(t, "id")
	defer tc.Cleanup()
	idUI, result, err := runIdentify(&tc, "t_charlie")
	require.NoError(t, err)
	checkCharlieProofs(t, idUI, &result.Upk.Current)
	checkDisplayKeys(t, idUI, 1, 1)
}

func TestIdDoug(t *testing.T) {
	tc := SetupEngineTest(t, "id")
	defer tc.Cleanup()
	idUI, result, err := runIdentify(&tc, "t_doug")
	require.NoError(t, err)
	checkDougProofs(t, idUI, &result.Upk.Current)
	checkDisplayKeys(t, idUI, 1, 1)
}

func TestIdEllen(t *testing.T) {
	tc := SetupEngineTest(t, "id")
	defer tc.Cleanup()
	idUI, _, err := runIdentify(&tc, "t_ellen")
	require.NoError(t, err)
	checkDisplayKeys(t, idUI, 0, 0)
}

// TestIdPGPNotEldest creates a user with a pgp key that isn't
// eldest key, then runs identify to make sure the pgp key is
// still displayed.
func TestIdPGPNotEldest(t *testing.T) {
	tc := SetupEngineTest(t, "id")
	defer tc.Cleanup()

	// create new user, then add pgp key
	u := CreateAndSignupFakeUser(tc, "login")
	uis := libkb.UIs{LogUI: tc.G.UI.GetLogUI(), SecretUI: u.NewSecretUI()}
	_, _, key := armorKey(t, tc, u.Email)
	eng, err := NewPGPKeyImportEngineFromBytes(tc.G, []byte(key), true)
	require.NoError(t, err)

	m := NewMetaContextForTest(tc).WithUIs(uis)
	err = RunEngine2(m, eng)
	require.NoError(t, err)

	Logout(tc)

	idUI, _, err := runIdentify(&tc, u.Username)
	require.NoError(t, err)

	checkDisplayKeys(t, idUI, 1, 1)
}

func TestIdGenericSocialProof(t *testing.T) {
	tc := SetupEngineTest(t, "id")
	defer tc.Cleanup()

	// create new user and have them prove a gubble.social account
	fu := CreateAndSignupFakeUser(tc, "login")
	_proveGubbleSocial(tc, fu, libkb.KeybaseSignatureV2, false /* promptPosted */)
	Logout(tc)

	fu2 := CreateAndSignupFakeUser(tc, "login")
	fu2.LoginOrBust(tc)

	idUI, result, err := runIdentify(&tc, fu.Username)
	require.NoError(t, err)

	// NOTE this will break once CORE-8658 is implemented
	require.Equal(t, keybase1.ProofResult{
		State:  keybase1.ProofState_TEMP_FAILURE,
		Status: keybase1.ProofStatus_BASE_HARD_ERROR,
		Desc:   "Not implemented",
	},
		idUI.ProofResults["gubble.social"].ProofResult,
	)

	checkKeyedProfile(t, idUI, &result.Upk.Current, fu.Username, map[string]string{
		"gubble.social": fu.Username,
	})
}

type FakeIdentifyUI struct {
	Proofs          map[string]string
	ProofResults    map[string]keybase1.LinkCheckResult
	User            *keybase1.User
	Keys            map[libkb.PGPFingerprint]*keybase1.TrackDiff
	DisplayKeyCalls int
	DisplayKeyDiffs []*keybase1.TrackDiff
	Outcome         *keybase1.IdentifyOutcome
	StartCount      int
	Token           keybase1.TrackToken
	BrokenTracking  bool
	DisplayTLFArg   keybase1.DisplayTLFCreateWithInviteArg
	DisplayTLFCount int
	FakeConfirm     bool
	sync.Mutex
}

func (ui *FakeIdentifyUI) FinishWebProofCheck(proof keybase1.RemoteProof, result keybase1.LinkCheckResult) error {
	ui.Lock()
	defer ui.Unlock()
	if ui.Proofs == nil {
		ui.Proofs = make(map[string]string)
	}
	ui.Proofs[proof.Key] = proof.Value

	if ui.ProofResults == nil {
		ui.ProofResults = make(map[string]keybase1.LinkCheckResult)
	}
	ui.ProofResults[proof.Key] = result
	if result.BreaksTracking {
		ui.BrokenTracking = true
	}
	return nil
}

func (ui *FakeIdentifyUI) FinishSocialProofCheck(proof keybase1.RemoteProof, result keybase1.LinkCheckResult) error {
	ui.Lock()
	defer ui.Unlock()
	if ui.Proofs == nil {
		ui.Proofs = make(map[string]string)
	}
	ui.Proofs[proof.Key] = proof.Value
	if ui.ProofResults == nil {
		ui.ProofResults = make(map[string]keybase1.LinkCheckResult)
	}
	ui.ProofResults[proof.Key] = result
	if result.BreaksTracking {
		ui.BrokenTracking = true
	}
	return nil
}

func (ui *FakeIdentifyUI) Confirm(outcome *keybase1.IdentifyOutcome) (result keybase1.ConfirmResult, err error) {
	ui.Lock()
	defer ui.Unlock()

	// Do a short sleep. This helps trigger bugs when other code is racing
	// against the UI here. (Note from Jack: In the bug I initially added this
	// for, 10ms was just enough to trigger it. I'm adding in an extra factor
	// of 10.)
	time.Sleep(100 * time.Millisecond)

	ui.Outcome = outcome
	bypass := ui.FakeConfirm || outcome.TrackOptions.BypassConfirm
	result.IdentityConfirmed = bypass
	result.RemoteConfirmed = bypass && !outcome.TrackOptions.ExpiringLocal
	return
}
func (ui *FakeIdentifyUI) DisplayCryptocurrency(keybase1.Cryptocurrency) error {
	return nil
}

func (ui *FakeIdentifyUI) DisplayKey(ik keybase1.IdentifyKey) error {
	ui.Lock()
	defer ui.Unlock()
	if ui.Keys == nil {
		ui.Keys = make(map[libkb.PGPFingerprint]*keybase1.TrackDiff)
	}

	fp := libkb.ImportPGPFingerprintSlice(ik.PGPFingerprint)
	if fp != nil {
		ui.Keys[*fp] = ik.TrackDiff
	}

	if ik.TrackDiff != nil {
		ui.DisplayKeyDiffs = append(ui.DisplayKeyDiffs, ik.TrackDiff)
	}

	ui.DisplayKeyCalls++
	return nil
}
func (ui *FakeIdentifyUI) ReportLastTrack(*keybase1.TrackSummary) error {
	return nil
}

func (ui *FakeIdentifyUI) Start(username string, _ keybase1.IdentifyReason, _ bool) error {
	ui.Lock()
	defer ui.Unlock()
	ui.StartCount++
	return nil
}

func (ui *FakeIdentifyUI) Cancel() error {
	return nil
}

func (ui *FakeIdentifyUI) Finish() error {
	return nil
}

func (ui *FakeIdentifyUI) Dismiss(_ string, _ keybase1.DismissReason) error {
	return nil
}

func (ui *FakeIdentifyUI) LaunchNetworkChecks(id *keybase1.Identity, user *keybase1.User) error {
	ui.Lock()
	defer ui.Unlock()
	ui.User = user
	return nil
}

func (ui *FakeIdentifyUI) DisplayTrackStatement(string) error {
	return nil
}

func (ui *FakeIdentifyUI) DisplayUserCard(keybase1.UserCard) error {
	return nil
}

func (ui *FakeIdentifyUI) ReportTrackToken(tok keybase1.TrackToken) error {
	ui.Token = tok
	return nil
}

func (ui *FakeIdentifyUI) SetStrict(b bool) {
}

func (ui *FakeIdentifyUI) DisplayTLFCreateWithInvite(arg keybase1.DisplayTLFCreateWithInviteArg) error {
	ui.DisplayTLFCount++
	ui.DisplayTLFArg = arg
	return nil
}
