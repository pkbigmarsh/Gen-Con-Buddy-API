package event

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestLoadEventCSV_DecodesWindows1252(t *testing.T) {
	// CSV with Windows-1252 encoded smart quotes and apostrophes:
	//   \x92 = right single quote (apostrophe)  →  '
	//   \x93 = left double quote               →  "
	//   \x94 = right double quote              →  "
	//   \x97 = em dash                         →  —
	csvContent := "game id,group,title,short description,long description,event type,game system,rules edition,minimum players,maximum players,age required,experience required,materials required,materials required details,start date & time,duration,end date & time,gm names,website,email,tournament?,round number,total rounds,minimum play time,attendee registration?,cost $,location,room name,table number,special category,tickets available,last modified\n" +
		"TST24XX000001,Test Group,Test Game,\"It\x92s a great game\",\"The GM said \x93hello\x94 \x97 enjoy\",ZED,None,None,1,4,Everyone (6+),None,No,,08/01/2024 09:00 AM,2,08/01/2024 11:00 AM,Test GM,,,No,1,1,2,Attendee Registration Required,$0,Hall A,Room 1,1,,4,08/01/2024\n"

	f, err := os.CreateTemp(t.TempDir(), "events_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.Write([]byte(csvContent)); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	events, err := LoadEventCSV(context.Background(), f.Name(), zerolog.Nop())
	if err != nil {
		t.Fatalf("LoadEventCSV returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	wantShort := "It\u2019s a great game"
	if events[0].ShortDescription != wantShort {
		t.Errorf("ShortDescription = %q, want %q", events[0].ShortDescription, wantShort)
	}

	wantLong := "The GM said \u201chello\u201d \u2014 enjoy"
	if events[0].LongDescription != wantLong {
		t.Errorf("LongDescription = %q, want %q", events[0].LongDescription, wantLong)
	}
}
