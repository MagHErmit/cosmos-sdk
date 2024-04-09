package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "cosmossdk.io/x/auth/types"

	codectestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestNewModuleCrendentials(t *testing.T) {
	ac := codectestutil.CodecOptions{}.GetAddressCodec()
	// wrong derivation keys
	_, err := authtypes.NewModuleCredential("group", []byte{})
	require.Error(t, err, "derivation keys must be non empty")
	_, err = authtypes.NewModuleCredential("group", [][]byte{{0x0, 0x30}, {}}...)
	require.Error(t, err)

	expected := sdk.MustAccAddressFromBech32("cosmos1fpn0w0yf4x300llf5r66jnfhgj4ul6cfahrvqsskwkhsw6sv84wsmz359y")

	_, err = authtypes.NewModuleCredential("group")
	require.NoError(t, err, "must be able to create a Root Module credential (see ADR-33)")

	credential, err := authtypes.NewModuleCredential("group", [][]byte{{0x20}, {0x0}}...)
	require.NoError(t, err)
	addr, err := sdk.AccAddressFromHexUnsafe(credential.Address().String())
	require.NoError(t, err)
	expectedAddr, err := ac.BytesToString(expected)
	require.NoError(t, err)
	addrStr, err := ac.BytesToString(addr)
	require.NoError(t, err)
	require.Equal(t, expectedAddr, addrStr)

	c, err := authtypes.NewModuleCredential("group", [][]byte{{0x20}, {0x0}}...)
	require.NoError(t, err)
	require.True(t, credential.Equals(c))

	c, err = authtypes.NewModuleCredential("group", [][]byte{{0x20}, {0x1}}...)
	require.NoError(t, err)
	require.False(t, credential.Equals(c))

	c, err = authtypes.NewModuleCredential("group", []byte{0x20})
	require.NoError(t, err)
	require.False(t, credential.Equals(c))
}

func TestNewBaseAccountWithPubKey(t *testing.T) {
	expected := sdk.MustAccAddressFromBech32("cosmos1fpn0w0yf4x300llf5r66jnfhgj4ul6cfahrvqsskwkhsw6sv84wsmz359y")

	credential, err := authtypes.NewModuleCredential("group", [][]byte{{0x20}, {0x0}}...)
	require.NoError(t, err)
	account, err := authtypes.NewBaseAccountWithPubKey(credential, codectestutil.CodecOptions{}.GetAddressCodec())
	require.NoError(t, err)
	require.Equal(t, expected, account.GetAddress())
	require.Equal(t, credential, account.GetPubKey())
}

func TestNewBaseAccountWithPubKey_WrongCredentials(t *testing.T) {
	_, err := authtypes.NewBaseAccountWithPubKey(cryptotypes.PubKey(nil), codectestutil.CodecOptions{}.GetAddressCodec())
	require.Error(t, err)
}
