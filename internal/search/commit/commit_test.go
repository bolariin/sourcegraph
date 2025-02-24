package commit

import (
	"bytes"
	"context"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"testing/quick"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git/gitapi"
)

func TestSearchCommitsInRepo(t *testing.T) {
	ctx := context.Background()
	db := new(dbtesting.MockDB)

	var calledVCSRawLogDiffSearch bool
	gitSignatureWithDate := gitapi.Signature{Date: time.Now().UTC().AddDate(0, 0, -1)}
	git.Mocks.RawLogDiffSearch = func(opt git.RawLogDiffSearchOptions) ([]*git.LogCommitSearchResult, bool, error) {
		calledVCSRawLogDiffSearch = true
		if want := "p"; opt.Query.Pattern != want {
			t.Errorf("got %q, want %q", opt.Query.Pattern, want)
		}
		if want := []string{
			"--no-prefix",
			"--max-count=" + strconv.Itoa(search.DefaultMaxSearchResults+1),
			"--unified=0",
			"--regexp-ignore-case",
			"rev",
		}; !reflect.DeepEqual(opt.Args, want) {
			t.Errorf("got %v, want %v", opt.Args, want)
		}
		return []*git.LogCommitSearchResult{
			{
				Commit: gitapi.Commit{ID: "c1", Author: gitSignatureWithDate},
				Diff:   &git.RawDiff{Raw: "x"},
			},
		}, true, nil
	}
	defer git.ResetMocks()

	q, err := query.ParseLiteral("p")
	if err != nil {
		t.Fatal(err)
	}
	repoRevs := &search.RepositoryRevisions{
		Repo: types.RepoName{ID: 1, Name: "repo"},
		Revs: []search.RevisionSpecifier{{RevSpec: "rev"}},
	}
	results, limitHit, timedOut, err := searchCommitsInRepo(ctx, db, search.CommitParameters{
		RepoRevs:    repoRevs,
		PatternInfo: &search.CommitPatternInfo{Pattern: "p", FileMatchLimit: int32(search.DefaultMaxSearchResults)},
		Query:       q,
		Diff:        true,
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []*result.CommitMatch{{
		Commit:      gitapi.Commit{ID: "c1", Author: gitSignatureWithDate},
		Repo:        types.RepoName{ID: 1, Name: "repo"},
		DiffPreview: &result.HighlightedString{Value: "x", Highlights: []result.HighlightedRange{}},
		Body:        result.HighlightedString{Value: "```diff\nx```", Highlights: []result.HighlightedRange{}},
	}}

	if !reflect.DeepEqual(results, want) {
		t.Errorf("results\ngot  %v\nwant %v", results, want)
	}

	wantDetail := "[`c1` one day ago](/repo/-/commit/c1)"
	if gotDetail := want[0].Detail(); gotDetail != wantDetail {
		t.Errorf("detail\ngot  %v\nwant %v", gotDetail, wantDetail)
	}

	wantLabel := "[repo](/repo) › [](/repo/-/commit/c1): [](/repo/-/commit/c1)"
	if gotLabel := want[0].Label(); gotLabel != wantLabel {
		t.Errorf("label\ngot  %v\nwant %v", gotLabel, wantLabel)
	}

	wantURL := "/repo/-/commit/c1"
	if gotURL := want[0].URL().String(); gotURL != wantURL {
		t.Errorf("url\ngot  %v\nwant %v", gotURL, wantURL)
	}

	if limitHit {
		t.Error("limitHit")
	}
	if timedOut {
		t.Error("timedOut")
	}
	if !calledVCSRawLogDiffSearch {
		t.Error("!calledVCSRawLogDiffSearch")
	}
}

func resetMocks() {
	database.Mocks = database.MockStores{}
	backend.Mocks = backend.MockServices{}
}

func TestExpandUsernamesToEmails(t *testing.T) {
	resetMocks()
	database.Mocks.Users.GetByUsername = func(ctx context.Context, username string) (*types.User, error) {
		if want := "alice"; username != want {
			t.Errorf("got %q, want %q", username, want)
		}
		return &types.User{ID: 123}, nil
	}
	database.Mocks.UserEmails.ListByUser = func(_ context.Context, opt database.UserEmailsListOptions) ([]*database.UserEmail, error) {
		if want := int32(123); opt.UserID != want {
			t.Errorf("got %v, want %v", opt.UserID, want)
		}
		t := time.Now()
		return []*database.UserEmail{
			{Email: "alice@example.com", VerifiedAt: &t},
			{Email: "alice@example.org", VerifiedAt: &t},
		}, nil
	}

	x, err := expandUsernamesToEmails(context.Background(), nil, []string{"foo", "@alice"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"foo", `alice@example\.com`, `alice@example\.org`}; !reflect.DeepEqual(x, want) {
		t.Errorf("got %q, want %q", x, want)
	}
}

func TestHighlightMatches(t *testing.T) {
	type args struct {
		pattern *regexp.Regexp
		data    []byte
	}
	tests := []struct {
		name string
		args args
		want *result.HighlightedString
	}{
		{
			// https://github.com/sourcegraph/sourcegraph/issues/4512
			name: "match at end",
			args: args{
				pattern: regexp.MustCompile(`白`),
				data:    []byte(`加一行空白`),
			},
			want: &result.HighlightedString{
				Value: "加一行空白",
				Highlights: []result.HighlightedRange{
					{
						Line:      1,
						Character: 4,
						Length:    1,
					},
				},
			},
		},
		{
			// https://github.com/sourcegraph/sourcegraph/issues/4512
			name: "two character match in middle",
			args: args{
				pattern: regexp.MustCompile(`行空`),
				data:    []byte(`加一行空白`),
			},
			want: &result.HighlightedString{
				Value: "加一行空白",
				Highlights: []result.HighlightedRange{
					{
						Line:      1,
						Character: 2,
						Length:    2,
					},
				},
			},
		},
		{
			// https://github.com/sourcegraph/sourcegraph/issues/4512
			name: "match at beginning",
			args: args{
				pattern: regexp.MustCompile(`加`),
				data:    []byte(`加一行空白`),
			},
			want: &result.HighlightedString{
				Value: "加一行空白",
				Highlights: []result.HighlightedRange{
					{
						Line:      1,
						Character: 0,
						Length:    1,
					},
				},
			},
		},

		{
			name: "invalid utf-8 ",
			args: args{
				pattern: regexp.MustCompile(`.`),
				data:    []byte("a\xc5z"),
			},
			want: &result.HighlightedString{
				Value: "a\xc5z",
				Highlights: []result.HighlightedRange{
					{
						Line:      1,
						Character: 0,
						Length:    1,
					},
					{
						Line:      1,
						Character: 1,
						Length:    1,
					},
					{
						Line:      1,
						Character: 2,
						Length:    1,
					},
				},
			},
		},

		{
			name: "multiline",
			args: args{
				pattern: regexp.MustCompile(`行`),
				data:    []byte("加一行空白\n加一空行白"),
			},
			want: &result.HighlightedString{
				Value: "加一行空白\n加一空行白",
				Highlights: []result.HighlightedRange{
					{
						Line:      1,
						Character: 2,
						Length:    1,
					},
					{
						Line:      2,
						Character: 3,
						Length:    1,
					},
				},
			},
		},

		// https://github.com/sourcegraph/sourcegraph/issues/4791
		{
			name: "unicode search that would be broken by tolower",
			args: args{
				pattern: regexp.MustCompile(`İ`),
				data:    []byte(`İi`),
			},
			want: &result.HighlightedString{
				Value: "İi",
				Highlights: []result.HighlightedRange{
					{
						Line:      1,
						Character: 0,
						Length:    1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := highlightMatches(tt.args.pattern, tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("highlightMatches() = %v, want %v", spew.Sdump(got), spew.Sdump(tt.want))
			}
		})
	}
}

func Benchmark_highlightMatches(b *testing.B) {
	as := bytes.Repeat([]byte{'a'}, 5000)
	lines := append(as, byte('\n'))
	lines = append(lines, as...)
	rx := regexp.MustCompile(`a`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = highlightMatches(rx, lines)
	}
}

// searchCommitsInRepo is a blocking version of searchCommitsInRepoStream.
func searchCommitsInRepo(ctx context.Context, db dbutil.DB, op search.CommitParameters) (results []*result.CommitMatch, limitHit, timedOut bool, err error) {
	var matches []result.Match
	err = searchCommitsInRepoStream(ctx, db, op, streaming.StreamFunc(func(event streaming.SearchEvent) {
		matches = append(matches, event.Results...)
		timedOut = timedOut || event.Stats.Status.Any(search.RepoStatusTimedout)
		limitHit = limitHit || event.Stats.Status.Any(search.RepoStatusLimitHit)
	}))
	for _, s := range matches {
		results = append(results, s.(*result.CommitMatch))
	}
	return results, limitHit, timedOut, err
}

func TestCommitSearchResult_Limit(t *testing.T) {
	f := func(nHighlights []int, limitInput uint32) bool {
		cr := &result.CommitMatch{
			Body: result.HighlightedString{
				Highlights: make([]result.HighlightedRange, len(nHighlights)),
			},
		}

		// It isn't interesting to test limit > ResultCount, so we bound it to
		// [1, ResultCount]
		count := cr.ResultCount()
		limit := (int(limitInput) % count) + 1

		after := cr.Limit(limit)
		newCount := cr.ResultCount()

		if after == 0 && newCount == limit {
			return true
		}

		t.Logf("failed limit=%d count=%d => after=%d newCount=%d", limit, count, after, newCount)
		return false
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error("quick check failed")
	}

	for nSymbols := 0; nSymbols <= 3; nSymbols++ {
		for limit := 0; limit <= nSymbols; limit++ {
			if !f(make([]int, nSymbols), uint32(limit)) {
				t.Error("small exhaustive check failed")
			}
		}
	}
}

func TestOrderedFuzzyRegexp(t *testing.T) {
	got := orderedFuzzyRegexp([]string{})
	if want := ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	got = orderedFuzzyRegexp([]string{"a"})
	if want := "a"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	got = orderedFuzzyRegexp([]string{"a", "b|c"})
	if want := "(a).*?(b|c)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_searchRangeToHighlights(t *testing.T) {
	type testCase struct {
		input      string
		inputRange result.Range
		output     []result.HighlightedRange
	}

	cases := []testCase{
		{
			input: "abc",
			inputRange: result.Range{
				Start: result.Location{Offset: 1, Line: 0, Column: 1},
				End:   result.Location{Offset: 2, Line: 0, Column: 2},
			},
			output: []result.HighlightedRange{{
				Line:      0,
				Character: 1,
				Length:    1,
			}},
		},
		{
			input: "abc\ndef\n",
			inputRange: result.Range{
				Start: result.Location{Offset: 2, Line: 0, Column: 2},
				End:   result.Location{Offset: 5, Line: 1, Column: 1},
			},
			output: []result.HighlightedRange{{
				Line:      0,
				Character: 2,
				Length:    1,
			}, {
				Line:      1,
				Character: 0,
				Length:    1,
			}},
		},
		{
			input: "abc\ndef\nghi\n",
			inputRange: result.Range{
				Start: result.Location{Offset: 0, Line: 0, Column: 0},
				End:   result.Location{Offset: 11, Line: 2, Column: 3},
			},
			output: []result.HighlightedRange{{
				Line:      0,
				Character: 0,
				Length:    3,
			}, {
				Line:      1,
				Character: 0,
				Length:    3,
			}, {
				Line:      2,
				Character: 0,
				Length:    3,
			}},
		},
	}

	for i, tc := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res := searchRangeToHighlights(tc.input, tc.inputRange)
			require.Equal(t, tc.output, res)
		})
	}
}
