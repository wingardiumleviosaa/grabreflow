package app

import "context"

type App interface {
	Context() context.Context
}
