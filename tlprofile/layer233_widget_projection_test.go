package tlprofile

import (
	"testing"

	"github.com/iamxvbaba/td/tg"
)

// A Layer 233 bot web widget has no wire identity before Layer 233. Rather than
// dropping the media (which reaches an older client as an empty text message) it
// must project to messageMediaUnsupported#9f84f49e for profiles 225-232.
func TestLayer233BotWidgetProjectsToUnsupportedInHistoricalProfiles(t *testing.T) {
	widget := &tg.MessageMediaWidget{Width: 320, Height: 200, Payload: []byte("<b>x</b>"), CanQuery: true}
	widget.SetWidgetID(7)
	message := func() *tg.Message {
		return &tg.Message{ID: 1, PeerID: &tg.PeerUser{UserID: 2}, Date: 3, Message: "widget", Media: widget}
	}

	for profile := Profile225; profile <= Profile232; profile++ {
		got, ok := layer232RoundTrip(t, profile, message()).(*tg.Message)
		if !ok {
			t.Fatalf("profile %d round trip returned %T", profile, got)
		}
		if _, ok := got.Media.(*tg.MessageMediaUnsupported); !ok {
			t.Fatalf("profile %d widget media = %T, want *tg.MessageMediaUnsupported", profile, got.Media)
		}
	}

	// The canonical profile keeps the widget unchanged.
	got, ok := layer232RoundTrip(t, Profile233, message()).(*tg.Message)
	if !ok {
		t.Fatalf("Layer 233 round trip returned %T", got)
	}
	if _, ok := got.Media.(*tg.MessageMediaWidget); !ok {
		t.Fatalf("Layer 233 widget media = %T, want *tg.MessageMediaWidget", got.Media)
	}
}
