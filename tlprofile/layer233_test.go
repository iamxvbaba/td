package tlprofile

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/iamxvbaba/td/bin"
	"github.com/iamxvbaba/td/tg"
)

func TestLayer233CanonicalProfile(t *testing.T) {
	if tg.Layer != 233 || ProfileCanonical != Profile233 {
		t.Fatalf("canonical Layer = tg:%d profile:%d, want 233", tg.Layer, ProfileCanonical)
	}
}

func layer233Channel(withNew bool) bin.Object {
	value := &tg.Channel{ID: 9, Title: "channel", Photo: &tg.ChatPhotoEmpty{}, Date: 1}
	if withNew {
		value.SetForumSidebar(true)
	}
	return value
}

func layer233User(withNew bool) bin.Object {
	value := &tg.User{ID: 4, FirstName: "bot"}
	if withNew {
		value.SetBotForumSidebar(true)
	}
	return value
}

func TestLayer233NewOptionalFlagsDoNotLeak(t *testing.T) {
	// channel#7ad15240 and user#b1b8cc83 keep their wire IDs. The new
	// forum-sidebar layout flags are canonical-only and must not reach 225-232.
	cases := []struct {
		name      string
		object    func(bool) bin.Object
		canonical uint32
	}{
		{"channel", layer233Channel, 0x7ad15240},
		{"user", layer233User, 0xb1b8cc83},
	}
	for _, tc := range cases {
		for profile := Profile225; profile <= Profile232; profile++ {
			plain := layer232Encode(t, profile, tc.object(false))
			rich := layer232Encode(t, profile, tc.object(true))
			if !bytes.Equal(plain, rich) {
				t.Fatalf("%s Layer 233 flag leaked into profile %d", tc.name, profile)
			}
		}
		canonical := layer232Encode(t, Profile233, tc.object(true))
		if id := binary32(canonical); id != tc.canonical {
			t.Fatalf("%s canonical id = %#08x, want %#08x", tc.name, id, tc.canonical)
		}
		want := tc.object(true)
		if got := layer232RoundTrip(t, Profile233, want); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s canonical round trip = %#v", tc.name, got)
		}
	}
}

func TestLayer233ToggleForumSidebarPolicy(t *testing.T) {
	request := &tg.ChannelsToggleForumRequest{
		Channel: &tg.InputChannelEmpty{},
		Enabled: true,
		Tabs:    true,
		Sidebar: true,
	}
	canonical := layer232Encode(t, Profile233, request)
	if id := binary32(canonical); id != 0xa2952b54 {
		t.Fatalf("canonical toggleForum id = %#08x, want 0xa2952b54", id)
	}
	if got := layer232RoundTrip(t, Profile233, request); !reflect.DeepEqual(got, request) {
		t.Fatalf("canonical toggleForum round trip = %#v", got)
	}
	// A historical request cannot express the sidebar layout. The
	// canonical-to-profile encoder drops the field; the profile-to-canonical
	// path admits the historical wire with sidebar=false. An old client only
	// knows the list and tabs layouts, and its tabs flag still chooses
	// between them.
	historical := &tg.ChannelsToggleForumRequest{Channel: &tg.InputChannelEmpty{}, Enabled: true, Tabs: true}
	for profile := Profile225; profile <= Profile232; profile++ {
		old := layer232Encode(t, profile, request)
		if id := binary32(old); id != 0x3ff75734 {
			t.Fatalf("profile %d toggleForum id = %#08x, want 0x3ff75734", profile, id)
		}
		if !bytes.Equal(old, layer232Encode(t, profile, historical)) {
			t.Fatalf("profile %d toggleForum encoding depends on sidebar", profile)
		}
		wire := &bin.Buffer{Buf: append([]byte(nil), old...)}
		if _, err := NewDispatcher().Admit(profile, wire, Limits{}); err != nil {
			t.Fatalf("profile %d rejected a historical toggleForum request: %v", profile, err)
		}
		if got := layer232RoundTrip(t, profile, request); !reflect.DeepEqual(got, historical) {
			t.Fatalf("profile %d historical toggleForum = %#v, want sidebar=false", profile, got)
		}
	}
}

func TestLayer233NewUpdatesDropInHistoricalProfiles(t *testing.T) {
	updates := []tg.UpdateClass{
		&tg.UpdateEditForumTopic{Peer: &tg.PeerUser{UserID: 1}, TopicID: 2},
		&tg.UpdateBotWidgetQuery{
			QueryID: 3, QueryType: &tg.WidgetQueryTypeData{}, UserID: 4, Peer: &tg.PeerUser{UserID: 1},
			MsgID: 5, ChatInstance: 6, ChatType: "private", Data: []byte{1},
		},
		&tg.UpdateInlineBotWidgetQuery{
			QueryID: 3, QueryType: &tg.WidgetQueryTypeData{}, UserID: 4,
			InlineMessageID: &tg.InputBotInlineMessageID{DCID: 1, ID: 2, AccessHash: 3},
			ChatInstance:    6, ChatType: "private", Data: []byte{1},
		},
		&tg.UpdateBotWidgetCopyQuery{QueryID: 3, QueryType: &tg.WidgetQueryTypeData{}, UserID: 4, Data: []byte{1}, WidgetID: 7},
		&tg.UpdateMessageWidgetData{Peer: &tg.PeerUser{UserID: 1}, MsgID: 2, Pts: 3, PtsCount: 1},
		&tg.UpdateTranscribedAudioFile{TranscriptionID: 1, Text: "hi"},
	}
	for _, update := range updates {
		var b bin.Buffer
		if err := EncodeObject(Profile233, update, &b); err != nil {
			t.Fatalf("canonical encode %T: %v", update, err)
		}
		decoded, err := DecodeObject(Profile233, &b, Limits{})
		if err != nil || !reflect.DeepEqual(decoded, update) {
			t.Fatalf("canonical round trip %T = %#v, %v", update, decoded, err)
		}
		for profile := Profile225; profile <= Profile232; profile++ {
			assert230EncodeRejected(t, profile, update)
			// Inside a container the update is dropped and its neighbours stay.
			known := &tg.UpdateConfig{}
			container := &tg.Updates{Updates: []tg.UpdateClass{update, known}, Users: []tg.UserClass{}, Chats: []tg.ChatClass{}, Date: 1, Seq: 1}
			got := layer232RoundTrip(t, profile, container).(*tg.Updates)
			if len(got.Updates) != 1 || !reflect.DeepEqual(got.Updates[0], known) {
				t.Fatalf("profile %d %T container = %#v, want only the known update", profile, update, got.Updates)
			}
		}
	}
}
