// Package quorum quorum algo
package quorum

import (
	"context"
	"errors"
)

var (
	errTooManyErrors = errors.New("too many errors")
	errUndetermined  = errors.New("undetermined")
)

// Vote vote interface
type Vote interface {
	Errored() bool
	Data() interface{}
}

func readQuorum(totalVotes int) int {
	return (totalVotes + 1) / 2
}

func writeQuorum(totalVotes int) int {
	return (totalVotes + 1) / 2
}

// HandleRead handle read process
func HandleRead(ctx context.Context, c chan Vote, totalVotes int) (interface{}, error) {
	var (
		quorum = readQuorum(totalVotes)
		nOK    = 0
		nNil   = 0
		nErr   = 0
	)

	for i := 0; i < cap(c); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case v := <-c:
			if v.Errored() {
				nErr++
				if nErr > totalVotes-quorum {
					return nil, errTooManyErrors
				}
			} else if data := v.Data(); data != nil {
				nOK++
				if nOK >= quorum {
					return data, nil
				}
			} else {
				nNil++
				if nNil > totalVotes-quorum {
					return nil, nil
				}
			}
		}
	}
	return nil, errUndetermined
}

// HandleWrite handle write process
func HandleWrite(ctx context.Context, c chan Vote, totalVotes int) error {
	var (
		quorum = writeQuorum(totalVotes)
		nErr   = 0
		nOK    = 0
	)
	for i := 0; i < cap(c); i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v := <-c:
			if v.Errored() {
				nErr++
				if nErr > totalVotes-quorum {
					return errTooManyErrors
				}
			} else {
				nOK++
				if nOK >= quorum {
					return nil
				}
			}
		}
	}
	return errUndetermined
}
