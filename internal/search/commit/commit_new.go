package commit

import (
	"bufio"
	"context"
	"regexp"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	gitprotocol "github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git/gitapi"
)

func searchInReposNew(ctx context.Context, db dbutil.DB, textParams *search.TextParametersForCommitParameters, params searchCommitsInReposParameters) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, repoRev := range textParams.Repos {
		// Skip the repo if no revisions were resolved for it
		if len(repoRev.Revs) == 0 {
			continue
		}

		rr := repoRev
		query := params.CommitParams.Query
		diff := params.CommitParams.Diff
		limit := int(textParams.PatternInfo.FileMatchLimit)

		args := &protocol.SearchRequest{
			Repo:        rr.Repo.Name,
			Revisions:   searchRevsToGitserverRevs(rr.Revs),
			Query:       gitprotocol.NewAnd(queryNodesToPredicates(query, query.IsCaseSensitive(), diff)...),
			IncludeDiff: diff,
			Limit:       limit,
		}

		onMatches := func(in []protocol.CommitMatch) {
			res := make([]result.Match, 0, len(in))
			for _, protocolMatch := range in {
				res = append(res, protocolMatchToCommitMatch(rr.Repo, diff, protocolMatch))
			}
			params.ResultChannel.Send(streaming.SearchEvent{
				Results: res,
			})
		}

		g.Go(func() error {
			limitHit, err := gitserver.DefaultClient.Search(ctx, args, onMatches)
			params.ResultChannel.Send(streaming.SearchEvent{
				Stats: streaming.Stats{
					IsLimitHit: limitHit,
				},
			})
			return err
		})
	}

	return g.Wait()
}

func searchRevsToGitserverRevs(in []search.RevisionSpecifier) []gitprotocol.RevisionSpecifier {
	out := make([]gitprotocol.RevisionSpecifier, 0, len(in))
	for _, rev := range in {
		out = append(out, gitprotocol.RevisionSpecifier{
			RevSpec:        rev.RevSpec,
			RefGlob:        rev.RefGlob,
			ExcludeRefGlob: rev.ExcludeRefGlob,
		})
	}
	return out
}

func queryNodesToPredicates(nodes []query.Node, caseSensitive, diff bool) []gitprotocol.Node {
	res := make([]gitprotocol.Node, 0, len(nodes))
	for _, node := range nodes {
		var newPred gitprotocol.Node
		switch v := node.(type) {
		case query.Operator:
			newPred = queryOperatorToPredicate(v, caseSensitive, diff)
		case query.Pattern:
			newPred = queryPatternToPredicate(v, caseSensitive, diff)
		case query.Parameter:
			newPred = queryParameterToPredicate(v, caseSensitive, diff)
		}
		if newPred != nil {
			res = append(res, newPred)
		}
	}
	return res
}

func queryOperatorToPredicate(op query.Operator, caseSensitive, diff bool) gitprotocol.Node {
	switch op.Kind {
	case query.And:
		return gitprotocol.NewAnd(queryNodesToPredicates(op.Operands, caseSensitive, diff)...)
	case query.Or:
		return gitprotocol.NewOr(queryNodesToPredicates(op.Operands, caseSensitive, diff)...)
	default:
		// I don't think we should have concats at this point, but ignore it if we do
		return nil
	}
}

func queryPatternToPredicate(pattern query.Pattern, caseSensitive, diff bool) gitprotocol.Node {
	patString := pattern.Value
	if pattern.Annotation.Labels.IsSet(query.Literal) {
		patString = regexp.QuoteMeta(pattern.Value)
	}

	var newPred gitprotocol.Node
	if diff {
		newPred = &gitprotocol.DiffMatches{Expr: patString, IgnoreCase: !caseSensitive}
	} else {
		newPred = &gitprotocol.MessageMatches{Expr: patString, IgnoreCase: !caseSensitive}
	}

	if pattern.Negated {
		return gitprotocol.NewNot(newPred)
	}
	return newPred
}

func queryParameterToPredicate(parameter query.Parameter, caseSensitive, diff bool) gitprotocol.Node {
	var newPred gitprotocol.Node
	switch parameter.Field {
	case query.FieldAuthor:
		// TODO(@camdencheek) look up emails (issue #25180)
		newPred = &gitprotocol.AuthorMatches{Expr: parameter.Value, IgnoreCase: !caseSensitive}
	case query.FieldCommitter:
		newPred = &gitprotocol.CommitterMatches{Expr: parameter.Value, IgnoreCase: !caseSensitive}
	case query.FieldBefore:
		t, _ := query.ParseGitDate(parameter.Value, time.Now) // field already validated
		newPred = &gitprotocol.CommitBefore{Time: t}
	case query.FieldAfter:
		t, _ := query.ParseGitDate(parameter.Value, time.Now) // field already validated
		newPred = &gitprotocol.CommitAfter{Time: t}
	case query.FieldMessage:
		newPred = &gitprotocol.MessageMatches{Expr: parameter.Value, IgnoreCase: !caseSensitive}
	case query.FieldContent:
		if diff {
			newPred = &gitprotocol.DiffMatches{Expr: parameter.Value, IgnoreCase: !caseSensitive}
		} else {
			newPred = &gitprotocol.MessageMatches{Expr: parameter.Value, IgnoreCase: !caseSensitive}
		}
	case query.FieldFile:
		newPred = &gitprotocol.DiffModifiesFile{Expr: parameter.Value, IgnoreCase: !caseSensitive}
	case query.FieldLang:
		newPred = &gitprotocol.DiffModifiesFile{Expr: search.LangToFileRegexp(parameter.Value), IgnoreCase: true}
	}

	if parameter.Negated {
		return gitprotocol.NewNot(newPred)
	}
	return newPred
}

func protocolMatchToCommitMatch(repo types.RepoName, diff bool, in protocol.CommitMatch) *result.CommitMatch {
	var (
		matchBody       string
		matchHighlights []result.HighlightedRange
		diffPreview     *result.HighlightedString
	)

	if diff {
		matchBody = "```diff\n" + in.Diff.Content + "\n```"
		matchHighlights = searchRangesToHighlights(matchBody, in.Diff.MatchedRanges.Add(result.Location{Line: 1, Offset: len("```diff\n")}))
		diffPreview = &result.HighlightedString{
			Value:      in.Diff.Content,
			Highlights: searchRangesToHighlights(in.Diff.Content, in.Diff.MatchedRanges),
		}
	} else {
		matchBody = "```COMMIT_EDITMSG\n" + in.Message.Content + "\n```"
		matchHighlights = searchRangesToHighlights(matchBody, in.Message.MatchedRanges.Add(result.Location{Line: 1, Offset: len("```COMMIT_EDITMSG\n")}))
	}

	return &result.CommitMatch{
		Commit: gitapi.Commit{
			ID: in.Oid,
			Author: gitapi.Signature{
				Name:  in.Author.Name,
				Email: in.Author.Email,
				Date:  in.Author.Date,
			},
			Committer: &gitapi.Signature{
				Name:  in.Committer.Name,
				Email: in.Committer.Email,
				Date:  in.Committer.Date,
			},
			Message: gitapi.Message(in.Message.Content),
			Parents: in.Parents,
		},
		Repo: repo,
		MessagePreview: &result.HighlightedString{
			Value:      in.Message.Content,
			Highlights: searchRangesToHighlights(in.Message.Content, in.Message.MatchedRanges),
		},
		DiffPreview: diffPreview,
		Body: result.HighlightedString{
			Value:      matchBody,
			Highlights: matchHighlights,
		},
	}
}

func searchRangesToHighlights(s string, ranges []result.Range) []result.HighlightedRange {
	res := make([]result.HighlightedRange, 0, len(ranges))
	for _, r := range ranges {
		res = append(res, searchRangeToHighlights(s, r)...)
	}
	return res
}

// searchRangeToHighlight converts a Range (which can cross multiple lines)
// into HighlightedRange, which is scoped to one line. In order to do this
// correctly, we need the string that is being highlighted in order to identify
// line-end boundaries within multi-line ranges.
// TODO(camdencheek): push the Range format up the stack so we can be smarter about multi-line highlights.
func searchRangeToHighlights(s string, r result.Range) []result.HighlightedRange {
	var res []result.HighlightedRange

	// Use a scanner to handle \r?\n
	scanner := bufio.NewScanner(strings.NewReader(s[r.Start.Offset:r.End.Offset]))
	lineNum := r.Start.Line
	for scanner.Scan() {
		line := scanner.Text()

		character := 0
		if lineNum == r.Start.Line {
			character = r.Start.Column
		}

		length := len(line)
		if lineNum == r.End.Line {
			length = r.End.Column - character
		}

		if length > 0 {
			res = append(res, result.HighlightedRange{
				Line:      int32(lineNum),
				Character: int32(character),
				Length:    int32(length),
			})
		}

		lineNum++
	}

	return res
}
