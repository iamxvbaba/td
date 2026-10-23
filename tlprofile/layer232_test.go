package tlprofile

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/iamxvbaba/td/bin"
	"github.com/iamxvbaba/td/tg"
)

func TestLayer232CanonicalProfile(t *testing.T) {
	if tg.Layer != 232 || ProfileCanonical != Profile232 {
		t.Fatalf("canonical Layer = tg:%d profile:%d, want 232", tg.Layer, ProfileCanonical)
	}
}

func layer232Encode(t *testing.T, profile Profile, object bin.Object) []byte {
	t.Helper()
	var b bin.Buffer
	if err := EncodeObject(profile, object, &b); err != nil {
		t.Fatalf("%d encode %T: %v", profile, object, err)
	}
	return b.Copy()
}

func layer232RoundTrip(t *testing.T, profile Profile, object bin.Object) bin.Object {
	t.Helper()
	wire := &bin.Buffer{Buf: layer232Encode(t, profile, object)}
	got, err := DecodeObject(profile, wire, Limits{})
	if err != nil {
		t.Fatalf("%d decode %T: %v", profile, object, err)
	}
	if wire.Len() != 0 {
		t.Fatalf("%d decode %T left %d bytes", profile, object, wire.Len())
	}
	return got
}

func layer232Channel(withNew bool) *tg.Channel {
	value := &tg.Channel{ID: 9, Title: "channel", Photo: &tg.ChatPhotoEmpty{}, Date: 1}
	if withNew {
		value.SetInNews(true)
		value.SetPinnedMsgID(77)
	}
	return value
}

func layer232ChannelFull(withNew bool) *tg.ChannelFull {
	value := &tg.ChannelFull{ID: 9, ChatPhoto: &tg.PhotoEmpty{ID: 1}}
	if withNew {
		value.SetShareTheme(&tg.ShareThemeDefault{})
	}
	return value
}

func layer232UserFull(withNew bool) *tg.UserFull {
	value := &tg.UserFull{ID: 1, CommonChatsCount: 2}
	if withNew {
		value.SetWalletCard(tg.WalletCard{Address: "EQ", Balance: 5})
		value.SetShareTheme(&tg.ShareThemeDefault{})
	}
	return value
}

func layer232ReplyHeader(withNew bool) *tg.MessageReplyHeader {
	value := &tg.MessageReplyHeader{}
	value.SetReplyToMsgID(3)
	value.SetQuote(true)
	value.SetQuoteText("quoted")
	if withNew {
		value.SetQuoteRich([]tg.RichTextClass{&tg.TextBold{Text: &tg.TextPlain{Text: "quoted"}}})
	}
	return value
}

func layer232InputReply(withNew bool) *tg.InputReplyToMessage {
	value := &tg.InputReplyToMessage{ReplyToMsgID: 3}
	value.SetQuoteText("quoted")
	if withNew {
		value.SetQuoteRich([]tg.RichTextClass{&tg.TextBold{Text: &tg.TextPlain{Text: "quoted"}}})
	}
	return value
}

func layer232NftItem(withNew bool) *tg.WalletNftItem {
	value := &tg.WalletNftItem{Address: "EQ-nft", OwnerAddress: "EQ-owner", Index: "1"}
	if withNew {
		value.SetGift(&tg.StarGift{ID: 5, Sticker: &tg.DocumentEmpty{ID: 6}, Stars: 7})
	}
	return value
}

func TestLayer232ReshuffledConstructorOracles(t *testing.T) {
	cases := []struct {
		name   string
		object func(withNew bool) bin.Object
		ids    map[Profile]uint32 // missing profile: unavailable
	}{
		{"channel", func(n bool) bin.Object { return layer232Channel(n) }, map[Profile]uint32{
			Profile225: 0x1c32b11c, Profile226: 0x1c32b11c, Profile227: 0x1c32b11c, Profile228: 0xd49f34c6,
			Profile229: 0xd49f34c6, Profile230: 0xd49f34c6, Profile231: 0xd49f34c6, Profile232: 0x7ad15240}},
		{"channelFull", func(n bool) bin.Object { return layer232ChannelFull(n) }, map[Profile]uint32{
			Profile225: 0xe4e0b29d, Profile226: 0xa04e8d3a, Profile227: 0xa04e8d3a, Profile228: 0xa04e8d3a,
			Profile229: 0xa04e8d3a, Profile230: 0xa04e8d3a, Profile231: 0xa04e8d3a, Profile232: 0xc4c4dd40}},
		{"userFull", func(n bool) bin.Object { return layer232UserFull(n) }, map[Profile]uint32{
			Profile225: 0x6cbe645, Profile226: 0x6cbe645, Profile227: 0x6cbe645, Profile228: 0x6cbe645,
			Profile229: 0x6cbe645, Profile230: 0x6cbe645, Profile231: 0x6cbe645, Profile232: 0xb90c2afd}},
		{"messageReplyHeader", func(n bool) bin.Object { return layer232ReplyHeader(n) }, map[Profile]uint32{
			Profile225: 0x1b97dd66, Profile226: 0x1b97dd66, Profile227: 0x1b97dd66, Profile228: 0x1b97dd66,
			Profile229: 0x1b97dd66, Profile230: 0x1b97dd66, Profile231: 0x1b97dd66, Profile232: 0xddb5ab75}},
		{"inputReplyToMessage", func(n bool) bin.Object { return layer232InputReply(n) }, map[Profile]uint32{
			Profile225: 0x3bd4b7c2, Profile226: 0x3bd4b7c2, Profile227: 0x3bd4b7c2, Profile228: 0x3bd4b7c2,
			Profile229: 0x3bd4b7c2, Profile230: 0x3bd4b7c2, Profile231: 0x3bd4b7c2, Profile232: 0x75c1ad70}},
		{"wallet.nftItem", func(n bool) bin.Object { return layer232NftItem(n) }, map[Profile]uint32{
			Profile230: 0x3441fd1c, Profile231: 0x3441fd1c, Profile232: 0x173dfaf3}},
	}
	for _, tc := range cases {
		for profile := Profile225; profile <= Profile232; profile++ {
			t.Run(fmt.Sprintf("%s/%d", tc.name, profile), func(t *testing.T) {
				want, available := tc.ids[profile]
				if !available {
					assert230EncodeRejected(t, profile, tc.object(false))
					return
				}
				plain := layer232Encode(t, profile, tc.object(false))
				if id := binary32(plain); id != want {
					t.Fatalf("constructor = %#08x, want %#08x", id, want)
				}
				rich := layer232Encode(t, profile, tc.object(true))
				if profile < Profile232 {
					// Layer 232 additions are optional metadata: historical wire is
					// byte-identical to the value without them.
					if !bytes.Equal(plain, rich) {
						t.Fatalf("Layer 232 fields leaked into profile %d wire", profile)
					}
					return
				}
				if got := layer232RoundTrip(t, profile, tc.object(true)); !reflect.DeepEqual(got, tc.object(true)) {
					t.Fatalf("Layer 232 round trip = %#v", got)
				}
			})
		}
	}
}

func TestLayer232RichQuoteKeepsFlatFallback(t *testing.T) {
	// The codec drops quote_rich for 225–231; the server-owned canonical value
	// must carry the flat quote_text fallback, which the projection retains.
	for profile := Profile225; profile <= Profile231; profile++ {
		header := layer232RoundTrip(t, profile, layer232ReplyHeader(true)).(*tg.MessageReplyHeader)
		if text, ok := header.GetQuoteText(); !ok || text != "quoted" || !header.Quote {
			t.Fatalf("%d reply header lost flat quote: %#v", profile, header)
		}
		if _, ok := header.GetQuoteRich(); ok {
			t.Fatalf("%d reply header kept quote_rich", profile)
		}
		reply := layer232RoundTrip(t, profile, layer232InputReply(true)).(*tg.InputReplyToMessage)
		if text, ok := reply.GetQuoteText(); !ok || text != "quoted" {
			t.Fatalf("%d input reply lost flat quote: %#v", profile, reply)
		}
	}
}

func binary32(wire []byte) uint32 {
	if len(wire) < 4 {
		return 0
	}
	return uint32(wire[0]) | uint32(wire[1])<<8 | uint32(wire[2])<<16 | uint32(wire[3])<<24
}

func layer232HistoricalWithdrawalWire(password bin.Encoder) []byte {
	var b bin.Buffer
	b.PutID(0xd06e93a8)
	b.PutID(0x2085c238) // inputSavedStarGiftSlug
	b.PutString("gift")
	if err := password.Encode(&b); err != nil {
		panic(err)
	}
	return b.Copy()
}

func TestLayer232StarGiftWithdrawalTemporaryPassword(t *testing.T) {
	// Literal schema oracle: payments.getStarGiftWithdrawalUrl#c8ca817b
	// flags:# stargift:InputSavedStarGift tmp_password:bytes.
	request := &tg.PaymentsGetStarGiftWithdrawalURLRequest{
		Stargift:    &tg.InputSavedStarGiftSlug{Slug: "gift"},
		TmpPassword: []byte{1, 2, 3},
	}
	var oracle bin.Buffer
	oracle.PutID(0xc8ca817b)
	oracle.PutInt(0)
	oracle.PutID(0x2085c238)
	oracle.PutString("gift")
	oracle.PutBytes([]byte{1, 2, 3})
	if wire := layer232Encode(t, Profile232, request); !bytes.Equal(wire, oracle.Raw()) {
		t.Fatalf("Layer 232 withdrawal wire %x != oracle %x", wire, oracle.Raw())
	}
	if got := layer232RoundTrip(t, Profile232, request); !reflect.DeepEqual(got, request) {
		t.Fatalf("Layer 232 withdrawal round trip = %#v", got)
	}

	for profile := Profile225; profile <= Profile231; profile++ {
		// The temporary-password method does not exist before Layer 232: a
		// canonical request is never encoded as the legacy SRP challenge,
		// whether or not it carries a temporary password.
		for _, tmp := range [][]byte{nil, {}, {1}} {
			assert230EncodeRejected(t, profile, &tg.PaymentsGetStarGiftWithdrawalURLRequest{
				Stargift: &tg.InputSavedStarGiftSlug{Slug: "gift"}, TmpPassword: tmp,
			})
		}
	}

	// Each exact historical profile admits only its own wire ID.
	modern := &bin.Buffer{Buf: layer232Encode(t, Profile232, request)}
	before := modern.Copy()
	if _, err := NewDispatcher().Admit(Profile231, modern, Limits{}); err == nil || !bytes.Equal(before, modern.Raw()) {
		t.Fatalf("Layer 231 admitted or consumed the Layer 232 withdrawal: %v", err)
	}
}

func TestLayer232LegacyStarGiftWithdrawalKeepsSRPProof(t *testing.T) {
	// The historical constructor is promoted to its own method, so the SRP
	// proof reaches the server unchanged instead of being dropped or turned
	// into a temporary password.
	if tg.PaymentsGetStarGiftWithdrawalURLLegacyRequestTypeID != 0xd06e93a8 {
		t.Fatalf("legacy withdrawal wire ID = %#x", tg.PaymentsGetStarGiftWithdrawalURLLegacyRequestTypeID)
	}
	if _, name, ok := SemanticName(SemanticMethodPaymentsGetStarGiftWithdrawalURLLegacy); !ok || name != "payments.getStarGiftWithdrawalUrlLegacy" {
		t.Fatalf("legacy withdrawal semantic = %q, %v", name, ok)
	}
	for _, password := range []tg.InputCheckPasswordSRPClass{
		&tg.InputCheckPasswordSRP{SRPID: 1, A: []byte{2}, M1: []byte{3}},
		&tg.InputCheckPasswordEmpty{},
	} {
		want := &tg.PaymentsGetStarGiftWithdrawalURLLegacyRequest{
			Stargift: &tg.InputSavedStarGiftSlug{Slug: "gift"},
			Password: password,
		}
		wire := layer232HistoricalWithdrawalWire(password)
		for profile := Profile225; profile <= Profile232; profile++ {
			if got := layer232Encode(t, profile, want); !bytes.Equal(got, wire) {
				t.Fatalf("%d legacy withdrawal wire %x != literal %x", profile, got, wire)
			}
			in := &bin.Buffer{Buf: append([]byte(nil), wire...)}
			got, err := DecodeObject(profile, in, Limits{})
			if err != nil || in.Len() != 0 || !reflect.DeepEqual(got, want) {
				t.Fatalf("%d legacy withdrawal decode = %#v, left %d: %v", profile, got, in.Len(), err)
			}

			dispatcher := NewDispatcher()
			var handled bin.Object
			if err := dispatcher.Register(SemanticMethodPaymentsGetStarGiftWithdrawalURLLegacy, func(_ context.Context, request bin.Object) (any, error) {
				handled = request
				return &tg.PaymentsStarGiftWithdrawalURL{URL: "https://fragment.example/gift"}, nil
			}); err != nil {
				t.Fatal(err)
			}
			admission, err := dispatcher.Admit(profile, &bin.Buffer{Buf: append([]byte(nil), wire...)}, Limits{})
			if err != nil {
				t.Fatalf("%d legacy withdrawal admission: %v", profile, err)
			}
			if admission.Call().Method() != SemanticMethodPaymentsGetStarGiftWithdrawalURLLegacy || admission.Call().WireID() != 0xd06e93a8 {
				t.Fatalf("%d legacy withdrawal admitted as %#x/%#x", profile, admission.Call().Method(), admission.Call().WireID())
			}
			result, err := dispatcher.Dispatch(context.Background(), admission)
			if err != nil {
				t.Fatalf("%d legacy withdrawal dispatch: %v", profile, err)
			}
			if !reflect.DeepEqual(handled, want) {
				t.Fatalf("%d legacy withdrawal handler got %#v", profile, handled)
			}
			var out bin.Buffer
			if err := result.Encode(&out); err != nil {
				t.Fatalf("%d legacy withdrawal result: %v", profile, err)
			}
		}
	}
}
