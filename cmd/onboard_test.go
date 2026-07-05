package cmd

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
)

func TestPollUntil(t *testing.T) {
	now := time.Now()
	done := apiclient.OnboardingStatus{FirstSpanAt: &now}
	cases := []struct {
		name      string
		results   []apiclient.OnboardingStatus
		fetchErr  error
		wantErr   bool
		wantCalls int
	}{
		{name: "done on first call", results: []apiclient.OnboardingStatus{done}, wantCalls: 1},
		{name: "done on third call", results: []apiclient.OnboardingStatus{{}, {}, done}, wantCalls: 3},
		{name: "fetch error stops polling", fetchErr: errors.New("boom"), wantErr: true, wantCalls: 1},
		{name: "timeout when never done", results: []apiclient.OnboardingStatus{{}, {}}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			fetch := func(context.Context) (apiclient.OnboardingStatus, error) {
				if tc.fetchErr != nil {
					calls++
					return apiclient.OnboardingStatus{}, tc.fetchErr
				}
				i := min(calls, len(tc.results)-1)
				calls++
				return tc.results[i], nil
			}
			st, err := pollUntil(context.Background(), io.Discard, 20*time.Millisecond, time.Millisecond,
				fetch, func(st apiclient.OnboardingStatus) bool { return st.FirstSpanAt != nil })
			if tc.wantErr {
				if err == nil {
					t.Fatalf("pollUntil = nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("pollUntil failed: %v", err)
			}
			if st.FirstSpanAt == nil {
				t.Error("returned status has no first span")
			}
			if tc.wantCalls > 0 && calls != tc.wantCalls {
				t.Errorf("fetch called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestPollUntilCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fetch := func(context.Context) (apiclient.OnboardingStatus, error) {
		return apiclient.OnboardingStatus{}, nil
	}
	_, err := pollUntil(ctx, io.Discard, time.Minute, time.Millisecond,
		fetch, func(apiclient.OnboardingStatus) bool { return false })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
