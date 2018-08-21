package main

import (
	"os"
	"sort"

	"github.com/elastos/Elastos.ELA.Client.SideChain/cli/info"
	"github.com/elastos/Elastos.ELA.Client.SideChain/cli/wallet"
	"github.com/elastos/Elastos.ELA.Client.SideChain/cli/mine"
	"github.com/elastos/Elastos.ELA.Client.SideChain/log"
	cliLog "github.com/elastos/Elastos.ELA.Client.SideChain/cli/log"
	"github.com/urfave/cli"
)

var Version string

func init() {
	log.InitLog()
}

func main() {
	app := cli.NewApp()
	app.Name = "side-cli"
	app.Version = Version
	app.HelpName = "side-cli"
	app.Usage = "command line tool for ELA blockchain"
	app.UsageText = "side-cli [global options] command [command options] [args]"
	app.HideHelp = false
	app.HideVersion = false
	//commands
	app.Commands = []cli.Command{
		*cliLog.NewCommand(),
		*info.NewCommand(),
		*wallet.NewCommand(),
		*mine.NewCommand(),
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))

	app.Run(os.Args)
}
