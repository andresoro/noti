package speech

import (
	"context"
	"fmt"
	"time"

	"github.com/variadico/noti/cmd/noti/cli"
	"github.com/variadico/noti/cmd/noti/config"
	"github.com/variadico/noti/cmd/noti/run"
	"github.com/variadico/noti/say"
	"github.com/variadico/vbs"
)

var cmdDefault = &say.Notification{
	Voice: "Alex",
	Text:  "{{.Cmd}} done!",
	Rate:  200,
}

func ptrs(n *say.Notification) []interface{} {
	if n == nil {
		return nil
	}

	return []interface{}{
		&n.Text,
		&n.Voice,
		&n.Rate,
	}
}

type Command struct {
	flag cli.Flags
	v    vbs.Printer
	n    *say.Notification

	help     bool
	ktimeout string
	timeout  string
}

func (c *Command) Parse(args []string) error {
	return c.flag.Parse(args)
}

func (c *Command) Notify(stats run.Stats) error {
	conf, err := config.File()
	if err != nil {
		c.v.Println(err)
	} else {
		c.v.Println("Found config file")
	}

	fromFlags := new(say.Notification)

	if c.flag.Passed("rate") {
		fromFlags.Rate = c.n.Rate
	}
	if c.flag.Passed("message", "m") {
		fromFlags.Text = c.n.Text
	}
	if c.flag.Passed("rate") {
		fromFlags.Rate = c.n.Rate
	}

	c.v.Println("Evaluating")
	c.v.Printf("Default: %+v\n", cmdDefault)
	c.v.Printf("Config: %+v\n", conf.Speech)
	c.v.Printf("Flags: %+v\n", fromFlags)

	config.EvalFields(ptrs(cmdDefault), stats)
	config.EvalFields(ptrs(conf.Speech), stats)
	config.EvalFields(ptrs(fromFlags), stats)

	c.v.Println("Merging")
	merged := new(say.Notification)
	err = config.MergeFields(
		ptrs(merged),
		ptrs(cmdDefault),
		ptrs(conf.Speech),
		ptrs(fromFlags),
	)
	if err != nil {
		return err
	}
	c.v.Printf("Merge result: %+v\n", merged)

	c.v.Println("Sending notification")
	err = merged.Send()
	c.v.Println("Sent notification")
	return err
}

func (c *Command) Run() error {
	if c.help {
		fmt.Println(helpText)
		return nil
	}

	if c.ktimeout == "" && c.timeout == "" {
		c.v.Println("Executing command")
		return c.Notify(run.Exec(c.flag.Args()...))
	} else if c.ktimeout != "" {
		d, err := time.ParseDuration(c.ktimeout)
		if err != nil {
			return err
		}

		c.v.Println("Executing command with timeout")
		stats := run.ExecWithTimeout(d, c.flag.Args()...)
		return c.Notify(stats)
	}

	// -timeout was set!

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stats := run.ExecNotify(ctx, c.flag.Args()...)
	for s := range stats {
		fmt.Println(">>>>>>>> SENDING NOTI!")
		err := c.Notify(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewCommand() cli.NotifyCmd {
	cmd := &Command{
		flag: cli.NewFlags("speech"),
		v:    vbs.New(),
		n:    new(say.Notification),
	}

	cmd.flag.SetStrings(&cmd.n.Text, "m", "message", cmdDefault.Text)

	cmd.flag.SetString(&cmd.n.Voice, "voice", cmdDefault.Voice)
	cmd.flag.SetInt(&cmd.n.Rate, "rate", cmdDefault.Rate)

	cmd.flag.SetBool(&cmd.v.Verbose, "verbose", false)
	cmd.flag.SetBools(&cmd.help, "h", "help", false)

	cmd.flag.SetString(&cmd.ktimeout, "ktimeout", "")
	cmd.flag.SetString(&cmd.timeout, "timeout", "")

	return cmd
}