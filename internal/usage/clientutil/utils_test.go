// Copyright 2023 Upbound Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientutil

import (
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewUsageQueryIterator(t *testing.T) {
	type args struct {
		account   string
		startTime time.Time
		endTime   time.Time
		window    time.Duration
	}
	type want struct {
		iter *UsageQueryIterator
		err  error
	}
	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"59MinuteWindow": {
			reason: "A 59m window should return an error.",
			args: args{
				account:   "test-account",
				startTime: time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
				endTime:   time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
				window:    59 * time.Minute,
			},
			want: want{
				err: errors.New("window must be 1h or greater"),
			},
		},
		"1HourWindow": {
			reason: "A 1h window should be accepted.",
			args: args{
				account:   "test-account",
				startTime: time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
				endTime:   time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
				window:    time.Hour,
			},
			want: want{
				iter: &UsageQueryIterator{
					Account: "test-account",
					Cursor:  time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
					EndTime: time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
					Window:  time.Hour,
				},
			},
		},
		"24HourWindow": {
			reason: "A 24h window should be accepted.",
			args: args{
				account:   "test-account",
				startTime: time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
				endTime:   time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
				window:    24 * time.Hour,
			},
			want: want{
				iter: &UsageQueryIterator{
					Account: "test-account",
					Cursor:  time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
					EndTime: time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
					Window:  24 * time.Hour,
				},
			},
		},
		"30DayWindow": {
			reason: "A 30 day window should be accepted.",
			args: args{
				account:   "test-account",
				startTime: time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
				endTime:   time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
				window:    30 * 24 * time.Hour,
			},
			want: want{
				iter: &UsageQueryIterator{
					Account: "test-account",
					Cursor:  time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
					EndTime: time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
					Window:  30 * 24 * time.Hour,
				},
			},
		},
		"1HourPrecision": {
			reason: "Times and window should be truncated to the hour.",
			args: args{
				account:   "test-account",
				startTime: time.Date(2006, 5, 4, 3, 2, 1, 0, time.UTC),
				endTime:   time.Date(2006, 5, 4, 4, 2, 1, 0, time.UTC),
				window:    time.Hour + time.Minute,
			},
			want: want{
				iter: &UsageQueryIterator{
					Account: "test-account",
					Cursor:  time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
					EndTime: time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
					Window:  time.Hour,
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			iter, err := NewUsageQueryIterator(tc.args.account, tc.args.startTime, tc.args.endTime, tc.args.window)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nNewUsageQueryIterator(...): -want err, +got err:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.iter, iter); diff != "" {
				t.Errorf("\n%s\nNewUsageQueryIterator(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestUsageQueryIterator(t *testing.T) {
	type args struct {
		account string
		start   time.Time
		end     time.Time
		window  time.Duration
	}
	type iteration struct {
		// These fields are exported for cmp.Diff().
		StartOffset string
		EndOffset   string
		Start       time.Time
		End         time.Time
		Err         error
	}
	cases := map[string]struct {
		reason string
		args   args
		want   []iteration
	}{
		"3HourRange1HourWindow": {
			reason: "3h range divided into 1h windows.",
			args: args{
				account: "test-account",
				start:   time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
				end:     time.Date(2006, 5, 4, 6, 0, 0, 0, time.UTC),
				window:  time.Hour,
			},
			want: []iteration{
				{
					StartOffset: "account=test-account/date=2006-05-04/hour=03/",
					EndOffset:   "account=test-account/date=2006-05-04/hour=04/",
					Start:       time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
					End:         time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
				},
				{
					StartOffset: "account=test-account/date=2006-05-04/hour=04/",
					EndOffset:   "account=test-account/date=2006-05-04/hour=05/",
					Start:       time.Date(2006, 5, 4, 4, 0, 0, 0, time.UTC),
					End:         time.Date(2006, 5, 4, 5, 0, 0, 0, time.UTC),
				},
				{
					StartOffset: "account=test-account/date=2006-05-04/hour=05/",
					EndOffset:   "account=test-account/date=2006-05-04/hour=06/",
					Start:       time.Date(2006, 5, 4, 5, 0, 0, 0, time.UTC),
					End:         time.Date(2006, 5, 4, 6, 0, 0, 0, time.UTC),
				},
			},
		},
		"3HourRange2HourWindow": {
			reason: "3h range divided into 2h windows.",
			args: args{
				account: "test-account",
				start:   time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
				end:     time.Date(2006, 5, 4, 6, 0, 0, 0, time.UTC),
				window:  2 * time.Hour,
			},
			want: []iteration{
				{
					StartOffset: "account=test-account/date=2006-05-04/hour=03/",
					EndOffset:   "account=test-account/date=2006-05-04/hour=05/",
					Start:       time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
					End:         time.Date(2006, 5, 4, 5, 0, 0, 0, time.UTC),
				},
				{
					StartOffset: "account=test-account/date=2006-05-04/hour=05/",
					EndOffset:   "account=test-account/date=2006-05-04/hour=06/",
					Start:       time.Date(2006, 5, 4, 5, 0, 0, 0, time.UTC),
					End:         time.Date(2006, 5, 4, 6, 0, 0, 0, time.UTC),
				},
			},
		},
		"3HourRange4HourWindow": {
			reason: "3h range divided into 4h windows.",
			args: args{
				account: "test-account",
				start:   time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
				end:     time.Date(2006, 5, 4, 6, 0, 0, 0, time.UTC),
				window:  4 * time.Hour,
			},
			want: []iteration{
				{
					StartOffset: "account=test-account/date=2006-05-04/hour=03/",
					EndOffset:   "account=test-account/date=2006-05-04/hour=06/",
					Start:       time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
					End:         time.Date(2006, 5, 4, 6, 0, 0, 0, time.UTC),
				},
			},
		},
		"3DayRange1DayWindow": {
			reason: "3-day range divided into 1-day windows.",
			args: args{
				account: "test-account",
				start:   time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
				end:     time.Date(2006, 5, 7, 3, 0, 0, 0, time.UTC),
				window:  24 * time.Hour,
			},
			want: []iteration{
				{
					StartOffset: "account=test-account/date=2006-05-04/hour=03/",
					EndOffset:   "account=test-account/date=2006-05-05/hour=03/",
					Start:       time.Date(2006, 5, 4, 3, 0, 0, 0, time.UTC),
					End:         time.Date(2006, 5, 5, 3, 0, 0, 0, time.UTC),
				},
				{
					StartOffset: "account=test-account/date=2006-05-05/hour=03/",
					EndOffset:   "account=test-account/date=2006-05-06/hour=03/",
					Start:       time.Date(2006, 5, 5, 3, 0, 0, 0, time.UTC),
					End:         time.Date(2006, 5, 6, 3, 0, 0, 0, time.UTC),
				},
				{
					StartOffset: "account=test-account/date=2006-05-06/hour=03/",
					EndOffset:   "account=test-account/date=2006-05-07/hour=03/",
					Start:       time.Date(2006, 5, 6, 3, 0, 0, 0, time.UTC),
					End:         time.Date(2006, 5, 7, 3, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			iter, err := NewUsageQueryIterator(tc.args.account, tc.args.start, tc.args.end, tc.args.window)
			if err != nil {
				t.Fatalf("NewUsageQueryIterator() error: %s", err)
			}

			got := []iteration{}
			for iter.More() {
				startOffset, endOffset, start, end, err := iter.Next()
				got = append(got, iteration{StartOffset: startOffset, EndOffset: endOffset, Start: start, End: end, Err: err})
			}

			if diff := cmp.Diff(tc.want, got, test.EquateErrors(), cmpopts.IgnoreUnexported(storage.Query{})); diff != "" {
				t.Errorf("\n%s\nUsageQueryIterator output: -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}
