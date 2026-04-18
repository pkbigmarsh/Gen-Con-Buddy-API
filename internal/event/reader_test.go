package event

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	_, err = f.Write([]byte(csvContent))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	events, err := LoadEventCSV(context.Background(), f.Name(), zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, events, 1)

	require.Equal(t, "It\u2019s a great game", events[0].ShortDescription,
		"Windows-1252 \\x92 should decode to UTF-8 right single quotation mark")
	require.Equal(t, "The GM said \u201chello\u201d \u2014 enjoy", events[0].LongDescription,
		"Windows-1252 smart quotes and em dash should decode to proper UTF-8")

}
