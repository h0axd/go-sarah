/*
Package todo is an example of stateful command that let users input required arguments step by step in a conversational manner.

On each valid input, given argument is stashed to *args.
*args is passed around until all required arguments are filled.
*/
package todo

import (
	"fmt"
	"github.com/oklahomer/go-sarah"
	"github.com/oklahomer/go-sarah/log"
	"github.com/oklahomer/go-sarah/slack"
	"golang.org/x/net/context"
	"regexp"
	"strings"
	"time"
)

var matchPattern = regexp.MustCompile(`^\.todo`)

// DummyStorage is an empty struct that represents a permanent storage.
type DummyStorage struct {
}

// Save saves given todo settings to permanent storage.
func (s *DummyStorage) Save(senderKey string, description string, due time.Time) {
	// Write to storage
}

type args struct {
	description string
	due         time.Time
}

// BuildCommand builds todo command with given storage.
func BuildCommand(storage *DummyStorage) sarah.Command {
	return &command{
		storage: storage,
	}
}

type command struct {
	storage *DummyStorage
}

var _ sarah.Command = (*command)(nil)

func (cmd *command) Identifier() string {
	return "todo"
}

func (cmd *command) Execute(_ context.Context, input sarah.Input) (*sarah.CommandResponse, error) {
	stripped := sarah.StripMessage(matchPattern, input.Message())
	if stripped == "" {
		// If description is not given, let user proceed to input one.
		return slack.NewStringResponseWithNext("Please input thing to do", cmd.inputDesc), nil
	}

	args := &args{
		description: stripped,
	}
	next := func(c context.Context, i sarah.Input) (*sarah.CommandResponse, error) {
		return cmd.inputDate(c, i, args)
	}
	return slack.NewStringResponseWithNext("Please input due date in YYYY-MM-DD format", next), nil
}

func (cmd *command) InputExample() string {
	return ".todo buy milk"
}

func (cmd *command) Match(input sarah.Input) bool {
	return strings.HasPrefix(strings.TrimSpace(input.Message()), ".todo")
}

func (cmd *command) inputDesc(_ context.Context, input sarah.Input) (*sarah.CommandResponse, error) {
	description := strings.TrimSpace(input.Message())
	if description == "" {
		// If no description is provided, let user input.
		next := func(c context.Context, i sarah.Input) (*sarah.CommandResponse, error) {
			return cmd.inputDesc(c, i)
		}
		return slack.NewStringResponseWithNext("Please input thing to do.", next), nil
	}

	// Let user proceed to next step to input due date.
	next := func(c context.Context, i sarah.Input) (*sarah.CommandResponse, error) {
		args := &args{
			description: description,
		}
		return cmd.inputDate(c, i, args)
	}
	return slack.NewStringResponseWithNext("Input due date. YYYY-MM-DD", next), nil
}

func (cmd *command) inputDate(_ context.Context, input sarah.Input, args *args) (*sarah.CommandResponse, error) {
	date := strings.TrimSpace(input.Message())

	reinput := func(c context.Context, i sarah.Input) (*sarah.CommandResponse, error) {
		return cmd.inputDate(c, i, args)
	}
	if date == "" {
		// If no due date is provided, let user input.
		return slack.NewStringResponseWithNext("Please input due date.", reinput), nil
	}

	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return slack.NewStringResponseWithNext("Please input valid date. YYYY-MM-DD", reinput), nil
	}

	next := func(c context.Context, i sarah.Input) (*sarah.CommandResponse, error) {
		return cmd.inputTime(c, i, date, args)
	}
	return slack.NewStringResponseWithNext("Input due time in HH:MM format. N if not specified.", next), nil
}

func (cmd *command) inputTime(_ context.Context, input sarah.Input, validDate string, args *args) (*sarah.CommandResponse, error) {
	t := strings.TrimSpace(input.Message())

	reinput := func(c context.Context, i sarah.Input) (*sarah.CommandResponse, error) {
		return cmd.inputTime(c, i, validDate, args)
	}
	if t == "" {
		return slack.NewStringResponseWithNext("Please input due time.", reinput), nil
	}

	if strings.ToLower(t) == "n" {
		// If there is no due time, consider the last minute is the due time.
		t = "23:59"
	}

	_, err := time.Parse("15:04", t)
	if err != nil {
		return slack.NewStringResponseWithNext("Please input valid due time in HH:MM format.", reinput), nil
	}

	due, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", validDate, t))
	if err != nil {
		// Should not reach here since previous time parse succeeded.
		log.Error("Failed to parse due date: %s", err.Error())
		return slack.NewStringResponse("Fatal error occurred."), nil
	}

	args.due = due
	next := func(c context.Context, i sarah.Input) (*sarah.CommandResponse, error) {
		return cmd.confirm(c, i, args)
	}
	confirmMessage := fmt.Sprintf("TODO: %s. Due is %s\nIs this O.K.? Y/N", args.description, args.due.Format("2006-01-02 15:04"))
	return slack.NewStringResponseWithNext(confirmMessage, next), nil
}

func (cmd *command) confirm(_ context.Context, input sarah.Input, args *args) (*sarah.CommandResponse, error) {
	msg := strings.TrimSpace(input.Message())
	if msg != "" {
		msg = strings.ToLower(msg)
	}

	if msg == "y" {
		cmd.storage.Save(input.SenderKey(), args.description, args.due)
		return slack.NewStringResponse("Saved."), nil
	}

	if msg == "n" {
		return slack.NewStringResponse("Aborted."), nil
	}

	reinput := func(c context.Context, i sarah.Input) (*sarah.CommandResponse, error) {
		return cmd.confirm(c, i, args)
	}
	return slack.NewStringResponseWithNext("Please input Y or N.", reinput), nil
}
