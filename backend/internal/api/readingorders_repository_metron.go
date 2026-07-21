package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func (s *cblRepositorySyncer) resolveMissingComicFromMetron(ctx context.Context, book cblBook) (*Comic, error) {
	if s.metronClient == nil {
		return nil, errors.New("metron is not configured")
	}

	var candidates []metron.Issue
	comicVineID, hasComicVineID := cblComicVineID(book)
	var err error
	if hasComicVineID {
		candidates, err = s.metronClient.SearchIssuesByComicVineID(ctx, comicVineID)
		if err != nil {
			return nil, fmt.Errorf("search Metron by Comic Vine ID %d: %w", comicVineID, err)
		}
	}
	if len(candidates) == 0 {
		series := strings.TrimSpace(book.Series)
		number := strings.TrimSpace(book.Number)
		if series == "" || number == "" {
			return nil, nil
		}
		candidates, err = s.metronClient.SearchIssues(ctx, "", series, number)
		if err != nil {
			return nil, fmt.Errorf("search Metron for %s #%s: %w", series, number, err)
		}
	}
	candidates = uniqueMetronIssueCandidates(candidates)
	if len(candidates) == 0 {
		return nil, nil
	}

	selected := candidates[0]
	if len(candidates) > 1 {
		selected, err = s.waitForMetronIssueResolution(ctx, book, comicVineID, candidates)
		if err != nil {
			return nil, err
		}
		if selected.ID == 0 {
			return nil, nil
		}
	}

	detail, err := s.metronClient.GetIssue(ctx, selected.ID)
	if err != nil {
		return nil, fmt.Errorf("load Metron issue %d: %w", selected.ID, err)
	}
	output, err := importMetronComicWithOptions(ctx, s.db, s.metronClient, s.covers, *detail, MetronImportOptions{Mode: "quick"})
	if err != nil {
		return nil, err
	}
	comic := output.Body.Comic
	return &comic, nil
}

func uniqueMetronIssueCandidates(issues []metron.Issue) []metron.Issue {
	seen := map[int]bool{}
	unique := make([]metron.Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.ID <= 0 || seen[issue.ID] {
			continue
		}
		seen[issue.ID] = true
		unique = append(unique, issue)
	}
	return unique
}

func (s *cblRepositorySyncer) waitForMetronIssueResolution(ctx context.Context, book cblBook, comicVineID int, candidates []metron.Issue) (metron.Issue, error) {
	byID := make(map[int]metron.Issue, len(candidates))
	items := make([]CBLMetronIssueCandidate, 0, len(candidates))
	for _, issue := range candidates {
		byID[issue.ID] = issue
		number := issue.Issue
		if number == "" {
			number = issue.Number
		}
		items = append(items, CBLMetronIssueCandidate{
			ID:          issue.ID,
			ComicVineID: issue.ComicVineID,
			Series:      issue.Series,
			SeriesYear:  issue.SeriesYear,
			Number:      number,
			Publisher:   issue.Publisher,
			CoverDate:   issue.CoverDate,
			CoverImage:  issue.CoverImage,
		})
	}

	s.mu.Lock()
	s.nextResolutionID++
	resolutionID := fmt.Sprintf("cbl-metron-%d", s.nextResolutionID)
	choices := s.resolutionChoices
	s.status.PendingResolution = &CBLMetronIssueResolution{
		ID:          resolutionID,
		Series:      strings.TrimSpace(book.Series),
		Number:      strings.TrimSpace(book.Number),
		Volume:      strings.TrimSpace(book.Volume),
		Year:        strings.TrimSpace(book.Year),
		ComicVineID: comicVineID,
		Candidates:  items,
	}
	s.mu.Unlock()
	s.broadcast()
	if choices == nil {
		return metron.Issue{}, errors.New("metron issue selection is unavailable")
	}

	for {
		select {
		case <-ctx.Done():
			return metron.Issue{}, ctx.Err()
		case choice := <-choices:
			if choice.ResolutionID != resolutionID {
				continue
			}
			if choice.MetronIssueID == 0 {
				return metron.Issue{}, nil
			}
			issue, ok := byID[choice.MetronIssueID]
			if !ok {
				return metron.Issue{}, errors.New("selected Metron issue is not a candidate")
			}
			return issue, nil
		}
	}
}

func (s *cblRepositorySyncer) resolveMetronIssue(resolutionID string, metronIssueID int) error {
	s.mu.Lock()
	if !s.status.Running || s.status.PendingResolution == nil || s.status.PendingResolution.ID != resolutionID {
		s.mu.Unlock()
		return errors.New("metron issue selection is no longer pending")
	}
	if metronIssueID != 0 {
		valid := false
		for _, candidate := range s.status.PendingResolution.Candidates {
			valid = valid || candidate.ID == metronIssueID
		}
		if !valid {
			s.mu.Unlock()
			return errors.New("selected Metron issue is not a candidate")
		}
	}
	choices := s.resolutionChoices
	s.status.PendingResolution = nil
	s.mu.Unlock()
	if choices == nil {
		return errors.New("metron issue selection is no longer pending")
	}
	choices <- cblMetronResolutionChoice{ResolutionID: resolutionID, MetronIssueID: metronIssueID}
	s.broadcast()
	return nil
}
