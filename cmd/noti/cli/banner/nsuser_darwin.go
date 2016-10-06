package banner

import (
	"context"
	"fmt"
	"time"

	"github.com/variadico/noti/cmd/noti/cli"
	"github.com/variadico/noti/cmd/noti/config"
	"github.com/variadico/noti/cmd/noti/run"
	"github.com/variadico/noti/nsuser"
	"github.com/variadico/vbs"
)

var cmdDefault = &nsuser.Notification{
	Title:           "{{.Cmd}}",
	InformativeText: "Done!",
	SoundName:       "Ping",
}

func ptrs(n *nsuser.Notification) []interface{} {
	return []interface{}{
		&n.Title,
		&n.Subtitle,
		&n.InformativeText,
		&n.ContentImage,
		&n.SoundName,
	}
}

type Command struct {
	flag cli.Flags
	v    vbs.Printer
	n    *nsuser.Notification

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

	fromFlags := new(nsuser.Notification)

	if c.flag.Passed("title", "t") {
		fromFlags.Title = c.n.Title
	}
	if c.flag.Passed("subtitle") {
		fromFlags.Subtitle = c.n.Subtitle
	}
	if c.flag.Passed("message", "m") {
		fromFlags.InformativeText = c.n.InformativeText
	}
	if c.flag.Passed("icon") {
		fromFlags.ContentImage = c.n.ContentImage
	}
	if c.flag.Passed("sound") {
		fromFlags.SoundName = c.n.SoundName
	}

	c.v.Println("Evaluating")
	c.v.Printf("Default: %+v\n", cmdDefault)
	c.v.Printf("Config: %+v\n", conf.Banner)
	c.v.Printf("Flags: %+v\n", fromFlags)

	config.EvalFields(ptrs(cmdDefault), stats)
	config.EvalFields(ptrs(conf.Banner), stats)
	config.EvalFields(ptrs(fromFlags), stats)

	c.v.Println("Merging")
	merged := new(nsuser.Notification)
	err = config.MergeFields(
		ptrs(merged),
		ptrs(cmdDefault),
		ptrs(conf.Banner),
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

	// Maybe we don't want to kill the process after the timeout.
	// Maybe just send the notification, but keep the process running.

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
		flag: cli.NewFlags("banner"),
		v:    vbs.New(),
		n:    new(nsuser.Notification),
	}

	cmd.flag.SetStrings(&cmd.n.Title, "t", "title", cmdDefault.Title)
	cmd.flag.SetStrings(&cmd.n.InformativeText, "m", "message", cmdDefault.InformativeText)

	cmd.flag.SetString(&cmd.n.Subtitle, "subtitle", cmdDefault.Subtitle)
	cmd.flag.SetString(&cmd.n.ContentImage, "icon", cmdDefault.ContentImage)
	cmd.flag.SetString(&cmd.n.SoundName, "sound", cmdDefault.SoundName)

	cmd.flag.SetBool(&cmd.v.Verbose, "verbose", false)
	cmd.flag.SetBools(&cmd.help, "h", "help", false)

	cmd.flag.SetString(&cmd.ktimeout, "ktimeout", "")
	cmd.flag.SetString(&cmd.timeout, "timeout", "")

	return cmd
}