package musicbot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/disgoorg/snowflake/v2"
)

type memoryRequests struct {
	requests    []SongRequest
	suggestions []SongSuggestion
}

func (m *memoryRequests) LastRequestAt(_ context.Context, userID snowflake.ID) (time.Time, error) {
	var last time.Time
	for _, r := range m.requests {
		if r.UserID == userID && r.CreatedAt.After(last) {
			last = r.CreatedAt
		}
	}
	return last, nil
}

func (m *memoryRequests) CountOpenRequests(_ context.Context, userID snowflake.ID) (int, error) {
	n := 0
	for _, r := range m.requests {
		if r.UserID == userID && r.Status == RequestQueued {
			n++
		}
	}
	return n, nil
}

func (m *memoryRequests) InsertRequest(_ context.Context, r *SongRequest) error {
	r.ID = int64(len(m.requests) + 1)
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	m.requests = append(m.requests, *r)
	return nil
}

func (m *memoryRequests) AdvanceRequests(context.Context, string, string) error { return nil }
func (m *memoryRequests) ExpireRequests(context.Context, time.Time) error       { return nil }
func (m *memoryRequests) RecentRequests(context.Context, int) ([]SongRequest, error) {
	return m.requests, nil
}
func (m *memoryRequests) UserRequests(_ context.Context, _ snowflake.ID, _ int) ([]SongRequest, error) {
	return m.requests, nil
}

func (m *memoryRequests) CountOpenSuggestions(_ context.Context, userID snowflake.ID) (int, error) {
	n := 0
	for _, s := range m.suggestions {
		if s.UserID == userID && (s.Status == SuggestionPending || s.Status == SuggestionReviewing) {
			n++
		}
	}
	return n, nil
}

func (m *memoryRequests) InsertSuggestion(_ context.Context, s *SongSuggestion) error {
	s.ID = int64(len(m.suggestions) + 1)
	m.suggestions = append(m.suggestions, *s)
	return nil
}

func (m *memoryRequests) UserSuggestions(context.Context, snowflake.ID, int) ([]SongSuggestion, error) {
	return m.suggestions, nil
}

func (m *memoryRequests) ListSuggestions(_ context.Context, status string, _ int) ([]SongSuggestion, error) {
	var out []SongSuggestion
	for _, s := range m.suggestions {
		if status == "" || s.Status == status {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *memoryRequests) GetSuggestion(_ context.Context, id int64) (SongSuggestion, error) {
	for _, s := range m.suggestions {
		if s.ID == id {
			return s, nil
		}
	}
	return SongSuggestion{}, ErrSuggestionNotFound
}

func (m *memoryRequests) UpdateSuggestion(_ context.Context, s SongSuggestion) error {
	for i := range m.suggestions {
		if m.suggestions[i].ID == s.ID {
			m.suggestions[i] = s
			return nil
		}
	}
	return ErrSuggestionNotFound
}

func (m *memoryRequests) DeleteOpenSuggestion(_ context.Context, id int64, userID snowflake.ID) (bool, error) {
	for i, s := range m.suggestions {
		if s.ID == id && s.UserID == userID && s.Status == SuggestionPending {
			m.suggestions = append(m.suggestions[:i], m.suggestions[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

type fakeSubmitter struct {
	calls []string
	err   error
}

func (f *fakeSubmitter) SubmitRequest(_ context.Context, station, uniqueID string) error {
	f.calls = append(f.calls, station+"/"+uniqueID)
	return f.err
}

type dmLog struct {
	messages map[snowflake.ID][]string
}

func (d *dmLog) send(_ context.Context, userID snowflake.ID, message string) error {
	d.messages[userID] = append(d.messages[userID], message)
	return nil
}

func newTestRequests(t *testing.T, cfg RequestConfig) (*Requests, *memoryRequests, *fakeSubmitter, *dmLog) {
	t.Helper()
	store := &memoryRequests{}
	submitter := &fakeSubmitter{}
	dms := &dmLog{messages: map[snowflake.ID][]string{}}
	r := NewRequests(cfg, store, submitter, testLibrary(t), dms.send, func(p string) string { return "https://chilly.test" + p })
	return r, store, submitter, dms
}

func TestSubmitRequestEnforcesLimits(t *testing.T) {
	r, store, submitter, _ := newTestRequests(t, RequestConfig{Cooldown: 5 * time.Minute, MaxPending: 2})
	now := time.Now()
	r.now = func() time.Time { return now }
	ctx := context.Background()

	req, err := r.Submit(ctx, 1, 9, "lib:chill:1", "web")
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != RequestQueued || req.Station != "chill" || req.Title != "Sunset Lover" || len(submitter.calls) != 1 {
		t.Fatalf("unexpected request %+v, calls %v", req, submitter.calls)
	}

	var limit *LimitError
	if _, err := r.Submit(ctx, 1, 9, "lib:chill:2", "web"); !errors.As(err, &limit) || !strings.Contains(limit.Message, "5 minutes") {
		t.Errorf("cooldown not enforced: %v", err)
	}

	r.now = func() time.Time { return now.Add(6 * time.Minute) }
	store.requests[0].CreatedAt = now.Add(-10 * time.Minute)
	store.requests = append(store.requests, SongRequest{UserID: 1, Status: RequestQueued, CreatedAt: now.Add(-20 * time.Minute)})
	if _, err := r.Submit(ctx, 1, 9, "lib:chill:2", "web"); !errors.As(err, &limit) || !strings.Contains(limit.Message, "2 requests waiting") {
		t.Errorf("pending limit not enforced: %v", err)
	}

	if _, err := r.Submit(ctx, 2, 9, "lib:chill:999", "web"); !errors.Is(err, ErrSelectionExpired) {
		t.Errorf("unknown songs should be rejected, got %v", err)
	}
}

func TestSubmitRequestPassesAzuraCastErrorsThrough(t *testing.T) {
	r, store, submitter, _ := newTestRequests(t, RequestConfig{})
	submitter.err = &azuracast.RequestError{Status: 400, Message: "This song was already requested and will play soon."}
	_, err := r.Submit(context.Background(), 1, 0, "lib:chill:1", "discord")
	if err == nil || err.Error() != "This song was already requested and will play soon." {
		t.Fatalf("got %v", err)
	}
	if len(store.requests) != 0 {
		t.Error("refused requests must not be stored")
	}
}

func TestSuggestValidatesAndDetectsExistingSongs(t *testing.T) {
	r, store, _, _ := newTestRequests(t, RequestConfig{MaxOpenSuggestions: 1})
	ctx := context.Background()

	var inLibrary *AlreadyInLibraryError
	if _, err := r.Suggest(ctx, 1, "Petit Biscuit", "Sunset Lover", "", ""); !errors.As(err, &inLibrary) || inLibrary.Track.Title != "Sunset Lover" {
		t.Errorf("songs already in the library should be caught, got %v", err)
	}
	var limit *LimitError
	if _, err := r.Suggest(ctx, 1, "", "Title", "", ""); !errors.As(err, &limit) {
		t.Errorf("missing artist should be rejected, got %v", err)
	}
	if _, err := r.Suggest(ctx, 1, "A", "B", "not a link", ""); !errors.As(err, &limit) {
		t.Errorf("bad links should be rejected, got %v", err)
	}

	s, err := r.Suggest(ctx, 1, " Tame Impala ", "The Less I Know The Better", "https://example.com/song", "love this")
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != SuggestionPending || s.Artist != "Tame Impala" || len(store.suggestions) != 1 {
		t.Errorf("unexpected suggestion %+v", s)
	}
	if _, err := r.Suggest(ctx, 1, "Another", "Song", "", ""); !errors.As(err, &limit) {
		t.Errorf("open suggestion limit not enforced, got %v", err)
	}
	if _, err := r.Suggest(ctx, 2, "Another", "Song", "", ""); err != nil {
		t.Errorf("limits are per user: %v", err)
	}
}

func TestMatchSuggestionsMarksAddedAndNotifies(t *testing.T) {
	r, store, _, dms := newTestRequests(t, RequestConfig{})
	store.suggestions = []SongSuggestion{
		{ID: 1, UserID: 7, Artist: "Joyner Lucas", Title: "Devil's Work", Status: SuggestionReviewing},
		{ID: 2, UserID: 8, Artist: "Nobody", Title: "Unreleased", Status: SuggestionPending},
		{ID: 3, UserID: 9, Artist: "M83", Title: "Midnight City", Status: SuggestionDeclined},
	}

	matched, err := r.MatchSuggestions(context.Background())
	if err != nil || matched != 1 {
		t.Fatalf("matched %d, err %v", matched, err)
	}
	if store.suggestions[0].Status != SuggestionAdded || store.suggestions[0].LibraryKey != "lib:hiphop:11" {
		t.Errorf("suggestion not marked added: %+v", store.suggestions[0])
	}
	if store.suggestions[2].Status != SuggestionDeclined {
		t.Error("declined suggestions must stay declined")
	}
	if msgs := dms.messages[7]; len(msgs) != 1 || !strings.Contains(msgs[0], "now in the Chilly library") || !strings.Contains(msgs[0], "https://chilly.test/dashboard/requests") {
		t.Errorf("expected an added DM, got %v", msgs)
	}
	if len(dms.messages[8]) != 0 {
		t.Error("unmatched suggestions must not be notified")
	}
}

func TestReviewNotifiesOnlyOnFinalChanges(t *testing.T) {
	r, store, _, dms := newTestRequests(t, RequestConfig{})
	store.suggestions = []SongSuggestion{{ID: 1, UserID: 5, Artist: "A", Title: "B", Status: SuggestionPending}}
	ctx := context.Background()

	if _, err := r.Review(ctx, 1, SuggestionReviewing, "", "", 99); err != nil {
		t.Fatal(err)
	}
	if len(dms.messages[5]) != 0 {
		t.Error("moving to reviewing should not DM")
	}
	s, err := r.Review(ctx, 1, SuggestionDeclined, "Not a fit for our stations", "", 99)
	if err != nil || s.ReviewedBy != 99 {
		t.Fatalf("review failed: %+v %v", s, err)
	}
	if msgs := dms.messages[5]; len(msgs) != 1 || !strings.Contains(msgs[0], "Not a fit for our stations") {
		t.Errorf("declined DM should include the reason, got %v", msgs)
	}
	if _, err := r.Review(ctx, 1, "bogus", "", "", 99); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("invalid status accepted: %v", err)
	}
	if _, err := r.Review(ctx, 404, SuggestionAdded, "", "", 99); !errors.Is(err, ErrSuggestionNotFound) {
		t.Errorf("missing suggestion: %v", err)
	}
}

func TestWithdrawOnlyPendingOwnSuggestions(t *testing.T) {
	r, store, _, _ := newTestRequests(t, RequestConfig{})
	store.suggestions = []SongSuggestion{
		{ID: 1, UserID: 5, Status: SuggestionPending},
		{ID: 2, UserID: 5, Status: SuggestionReviewing},
	}
	ctx := context.Background()
	if err := r.Withdraw(ctx, 1, 6); !errors.Is(err, ErrSuggestionNotFound) {
		t.Error("users must not withdraw other people's suggestions")
	}
	if err := r.Withdraw(ctx, 2, 5); !errors.Is(err, ErrSuggestionNotFound) {
		t.Error("suggestions under review can't be withdrawn")
	}
	if err := r.Withdraw(ctx, 1, 5); err != nil || len(store.suggestions) != 1 {
		t.Errorf("withdraw failed: %v", err)
	}
}

func TestFormatWait(t *testing.T) {
	cases := map[time.Duration]string{
		30 * time.Second:               "31 seconds",
		time.Minute:                    "1 minute",
		4*time.Minute + 10*time.Second: "5 minutes",
	}
	for d, want := range cases {
		if got := formatWait(d); got != want {
			t.Errorf("formatWait(%s) = %q, want %q", d, got, want)
		}
	}
}
