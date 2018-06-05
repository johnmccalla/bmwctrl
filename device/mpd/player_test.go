package mpd

import (
	"fmt"
	"testing"

	"github.com/oandrew/ipod/lingo-extremote"
)

func TestPlayer(t *testing.T) {

}

func TestRetrieveCategorizedDatabaseRecords(t *testing.T) {
	p := NewPlayer(nil)
	a := p.RetrieveCategorizedDatabaseRecords(extremote.DbCategoryArtist, 1, 2)
	fmt.Print(a)
}

func TestSelectDBRecord(t *testing.T) {
	p := NewPlayer(nil)
	p.SelectDBRecord(extremote.DbCategoryArtist, 4)
	p.PlayCurrentSelection(0)
}
