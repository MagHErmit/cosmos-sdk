package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	cosmos_proto "github.com/cosmos/cosmos-proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"cosmossdk.io/core/transaction"
	"cosmossdk.io/math"
	"cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/aminojson"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/tests/integration/tx/internal/pulsar/testpb"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	gogoproto "github.com/cosmos/gogoproto/proto"
)

func ProvideCustomGetSigner() signing.CustomGetSigner {
	return signing.CustomGetSigner{
		MsgType: proto.MessageName(&testpb.TestRepeatedFields{}),
		Fn: func(msg proto.Message) ([][]byte, error) {
			testMsg := msg.(*testpb.TestRepeatedFields)
			// arbitrary logic
			signer := testMsg.NullableDontOmitempty[1].Value
			return [][]byte{[]byte(signer)}, nil
		},
	}
}

type noOpAddressCodec struct{}

func (a noOpAddressCodec) StringToBytes(text string) ([]byte, error) {
	return []byte(text), nil
}

func (a noOpAddressCodec) BytesToString(bz []byte) (string, error) {
	return string(bz), nil
}

type SigningFixture struct {
	txConfig   client.TxConfig
	legacy     *codec.LegacyAmino
	protoCodec *codec.ProtoCodec
	options    SigningFixtureOptions
	registry   codectypes.InterfaceRegistry
}

type SigningFixtureOptions struct {
	DoNotSortFields bool
}

func NewSigningFixture(
	t *testing.T,
	options SigningFixtureOptions,
	modules ...module.AppModule,
) *SigningFixture {
	t.Helper()
	// set up transaction and signing infra
	addressCodec, valAddressCodec := noOpAddressCodec{}, noOpAddressCodec{}
	customGetSigners := []signing.CustomGetSigner{ProvideCustomGetSigner()}
	interfaceRegistry, _, err := codec.ProvideInterfaceRegistry(
		addressCodec,
		valAddressCodec,
		customGetSigners,
	)
	require.NoError(t, err)
	protoCodec := codec.ProvideProtoCodec(interfaceRegistry)
	signingOptions := &signing.Options{
		FileResolver:          interfaceRegistry,
		AddressCodec:          addressCodec,
		ValidatorAddressCodec: valAddressCodec,
	}
	for _, customGetSigner := range customGetSigners {
		signingOptions.DefineCustomGetSigners(customGetSigner.MsgType, customGetSigner.Fn)
	}
	txConfig, err := tx.NewTxConfigWithOptions(
		protoCodec,
		tx.ConfigOptions{
			EnabledSignModes: []signingtypes.SignMode{
				signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
			},
			SigningOptions: signingOptions,
		})
	require.NoError(t, err)

	legacyAminoCodec := codec.NewLegacyAmino()
	mb := module.NewManager(modules...)
	std.RegisterLegacyAminoCodec(legacyAminoCodec)
	std.RegisterInterfaces(interfaceRegistry)
	mb.RegisterLegacyAminoCodec(legacyAminoCodec)
	mb.RegisterInterfaces(interfaceRegistry)

	return &SigningFixture{
		txConfig:   txConfig,
		legacy:     legacyAminoCodec,
		options:    options,
		protoCodec: protoCodec,
		registry:   interfaceRegistry,
	}
}

func (s *SigningFixture) RequireLegacyAminoEquivalent(t *testing.T, msg transaction.Msg) {
	t.Helper()
	// create tx envelope
	txBuilder := s.txConfig.NewTxBuilder()
	err := txBuilder.SetMsgs([]types.Msg{msg}...)
	require.NoError(t, err)
	builtTx := txBuilder.GetTx()

	// round trip it to simulate application usage
	txBz, err := s.txConfig.TxEncoder()(builtTx)
	require.NoError(t, err)
	theTx, err := s.txConfig.TxDecoder()(txBz)
	require.NoError(t, err)

	// create signing envelope
	signerData := signing.SignerData{
		Address:       "sender-address",
		ChainID:       "test-chain",
		AccountNumber: 0,
		Sequence:      0,
	}
	adaptableTx, ok := theTx.(authsigning.V2AdaptableTx)
	require.True(t, ok)

	legacytx.RegressionTestingAminoCodec = s.legacy
	defer func() {
		legacytx.RegressionTestingAminoCodec = nil
	}()
	legacyAminoSignHandler := tx.NewSignModeLegacyAminoJSONHandler()
	legacyBz, err := legacyAminoSignHandler.GetSignBytes(
		signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
		authsigning.SignerData{
			ChainID:       signerData.ChainID,
			Address:       signerData.Address,
			AccountNumber: signerData.AccountNumber,
			Sequence:      signerData.Sequence,
		},
		theTx)
	require.NoError(t, err)

	handler := aminojson.NewSignModeHandler(aminojson.SignModeHandlerOptions{})
	signBz, err := handler.GetSignBytes(
		context.Background(),
		signerData,
		adaptableTx.GetSigningTxData(),
	)
	require.NoError(t, err)

	require.Truef(t,
		bytes.Equal(legacyBz, signBz),
		"legacy: %s\n  x/tx: %s", string(legacyBz), string(signBz))
}

func (s *SigningFixture) MarshalLegacyAminoJSON(t *testing.T, o any) []byte {
	t.Helper()
	bz, err := s.legacy.MarshalJSON(o)
	require.NoError(t, err)
	if s.options.DoNotSortFields {
		return bz
	}
	sortedBz, err := sortJson(bz)
	require.NoError(t, err)
	return sortedBz
}

func (s *SigningFixture) UnmarshalGogoProto(bz []byte, ptr transaction.Msg) error {
	return s.protoCodec.Unmarshal(bz, ptr)
}

func (s *SigningFixture) MessageDescriptor(t *testing.T, msg transaction.Msg) protoreflect.MessageDescriptor {
	t.Helper()
	typeName := gogoproto.MessageName(msg)
	msgDesc, err := s.registry.FindDescriptorByName(protoreflect.FullName(typeName))
	require.NoError(t, err)
	return msgDesc.(protoreflect.MessageDescriptor)
}

// DynamicMessage is identical to the Decoder implementation in
// https://github.com/cosmos/cosmos-sdk/blob/6d2f6ff068c81c5783e01319beaa51c7dbb43edd/x/tx/decode/decode.go#L136
// It is duplicated here to test dynamic message implementations specifically.
// The code path linked above is also covered in this package.
func (s *SigningFixture) DynamicMessage(t *testing.T, msg transaction.Msg) proto.Message {
	t.Helper()
	msgDesc := s.MessageDescriptor(t, msg)
	protoBz, err := gogoproto.Marshal(msg)
	require.NoError(t, err)
	dynamicMsg := dynamicpb.NewMessageType(msgDesc).New().Interface()
	err = proto.Unmarshal(protoBz, dynamicMsg)
	require.NoError(t, err)
	// err = postfixDynamicMessage(dynamicMsg.ProtoReflect())
	// require.NoError(t, err)
	return dynamicMsg
}

func postfixDynamicMessage(msg protoreflect.Message) error {
	fields := msg.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		v := msg.Get(field)

		switch val := v.Interface().(type) {
		case protoreflect.Message:
			err := postfixDynamicMessage(val)
			if err != nil {
				return err
			}
		case string:
			fieldOpts := field.Options()
			if proto.HasExtension(fieldOpts, cosmos_proto.E_Scalar) {
				scalar := proto.GetExtension(fieldOpts, cosmos_proto.E_Scalar).(string)
				switch scalar {
				case "cosmos.Dec":
					dec := &math.LegacyDec{}
					err := dec.Unmarshal([]byte(val))
					if err != nil {
						return err
					}
					decStr := dec.String()
					fmt.Printf("set %s=%s at %d\n", field.Name(), decStr, field.Number())
					//valStr := protoreflect.ValueOfString("5.400000000000000000")
					valStr := protoreflect.ValueOfString(decStr)
					msg.Set(field, valStr)
				}
			}
		}
	}
	return nil
}

// sortJson sorts the JSON bytes by way of the side effect of unmarshalling and remarshalling
// the JSON using encoding/json.  This hacky way of sorting JSON fields was used by the legacy
// amino JSON encoding x/auth/migrations/legacytx.StdSignBytes.  It is used here ensure the x/tx
// JSON encoding is equivalent to the legacy amino JSON encoding.
func sortJson(bz []byte) ([]byte, error) {
	var c any
	err := json.Unmarshal(bz, &c)
	if err != nil {
		return nil, err
	}
	js, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return js, nil
}
