package hello

import (
	"github.com/oklahomer/go-sarah"
	"github.com/oklahomer/go-sarah/slack"
	"golang.org/x/net/context"
	"regexp"
)

// SlackCommand provides default setup of hello command.
var SlackCommand = sarah.NewCommandBuilder().
	Identifier("hello").
	InputExample(".hello").
	MatchPattern(regexp.MustCompile(`\.hello`)).
	Func(func(_ context.Context, input sarah.Input) (*sarah.CommandResponse, error) {
		return slack.NewStringResponse("Hello!"), nil
	}).
	MustBuild()
