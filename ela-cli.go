package main

import (
	"os"
	"sort"

	"github.com/urfave/cli"

	"github.com/elastos/Elastos.ELA.Client.Core/cli/info"
	"github.com/elastos/Elastos.ELA.Client.Core/cli/wallet"
	"github.com/elastos/Elastos.ELA.Client.Core/cli/mine"
	"github.com/elastos/Elastos.ELA.Client.Core/common/log"
	cliLog "github.com/elastos/Elastos.ELA.Client.Core/cli/log"
	_ "github.com/elastos/Elastos.ELA.Client.SideChain/rpc"
	_ "github.com/elastos/Elastos.ELA.Client.SideChain/cli/wallet"
)

var Version string

func init() {
	log.InitLog()
}

func main() {
	app := cli.NewApp()
	app.Name = "ela-cli"
	app.Version = Version
	app.HelpName = "ela-cli"
	app.Usage = "command line tool for ELA blockchain"
	app.UsageText = "ela-cli [global options] command [command options] [args]"
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
