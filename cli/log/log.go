package log

import (
	"fmt"

	"github.com/elastos/Elastos.ELA.Client.SideChain/rpc"

	"github.com/urfave/cli"
)

func debugAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	if level := c.Int("level"); level >= 0 {
		result, err := rpc.CallAndUnmarshal("setloglevel", rpc.Param("level", level))
		if err != nil {
			fmt.Println("error: set debug info failed, ", err)
			return err
		}
		fmt.Println(result)
		return nil
	}

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{Name: "log",
		Usage: "set node log output level",
		Description: "With side-cli log, you could change node log output level.\n" +
			"\t levels are 0~6, the lower level the more logs will be print out, 0 means print out everything",
		ArgsUsage: "[args]",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "level, l",
				Usage: "--log level",
				Value: -1,
			},
		},
		Action: debugAction,
		OnUsageError: func(c *cli.Context, err error, isSubCommand bool) error {
			return cli.NewExitError(err, 1)
		},
	}
}
