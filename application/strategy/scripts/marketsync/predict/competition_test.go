package predict

import "testing"

func TestLeagueFromSportIDPrefixClubFriendlies(t *testing.T) {
	cfg := SportsConfig{
		CompetitionBySportID: map[string]string{"clf": "俱乐部友谊赛"},
		CompetitionEnByTag:     map[string]string{"clf": "Club Friendlies"},
	}
	cn, en, ok := leagueFromSportIDPrefix("clf-tur-jab-2026-11-17", cfg)
	if !ok || cn != "俱乐部友谊赛" || en != "Club Friendlies" {
		t.Fatalf("got %q %q ok=%v", cn, en, ok)
	}
}