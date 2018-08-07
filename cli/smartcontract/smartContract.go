package smartcontract

import (
	"os"
	"io/ioutil"
	"fmt"

	"github.com/elastos/Elastos.ELA.Client.SideChain/wallet"
	 wallet2 "github.com/elastos/Elastos.ELA.Client.SideChain/cli/wallet"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/urfave/cli"
)

func contractAction(context *cli.Context) error {
	if context.NumFlags() == 0 {
		cli.ShowSubcommandHelp(context)
		return nil
	}

	deploy := context.Bool("deploy")
	invoke := context.Bool("invoke")

	if !deploy && !invoke {
		fmt.Println("missing --deploy -d or --invoke -i")
		return nil
	}
	walletImpl, err := wallet.GetWallet()
	if err != nil {
		fmt.Println("error: open wallet failed, ", err)
		os.Exit(2)
	}
	walletName := context.String("wallet")
	password := context.String("password")
	password = "123";
	if (walletName == "") {
		walletName = "keystore.dat"
	}
	pwd := []byte(password)
	err = walletImpl.Open(walletName, pwd)
	if err != nil {
		fmt.Println("error: open wallet failed, ", err)
		os.Exit(2)
	}

	if deploy {
		codeStr := context.String("code")
		fileStr := context.String("file")

		if codeStr == "" && fileStr == "" {
			fmt.Println("missing args [--code] or [--file]")
			return nil
		}
		if codeStr != "" && fileStr != "" {
			fmt.Println("too many input args")
			return nil
		}
		if fileStr != "" {
			bytes, err := ioutil.ReadFile(fileStr)
			if err != nil {
				fmt.Println("read avm file err")
				return nil
			}
			codeStr = common.BytesToHexString(bytes)
		}

		err = wallet2.CreateDeployTransaction(context, walletImpl, codeStr)
		if err != nil {
			fmt.Println("error:", err)
			os.Exit(701)
		}
	}

	if invoke {
		err = wallet2.CreateInvokeTransaction(context, walletImpl)
		if err != nil {
			fmt.Println("error:", err)
			os.Exit(701)
		}
	}

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "contract",
		Usage:       "deploy or invoke your smartcontract",
		Description: "you could deploy or invoke your smartcontract.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "deploy, d",
				Usage: "deploy smartcontract",
			},
			cli.BoolFlag{
				Name:  "invoke, i",
				Usage: "invoke smartcontract",
			},
			cli.StringFlag{
				Name:  "code, c",
				Usage: "deploy contract code",
			},
			cli.StringFlag{
				Name:  "wallet, w",
				Usage: "wallet db name",
			},
			cli.StringFlag{
				Name:  "password, m",
				Usage: "wallet db password",
			},
			cli.StringFlag{
				Name:  "file, f",
				Usage: "deploy avm file",
			},
			cli.StringFlag{
				Name:  "fee",
				Usage: "the transfer fee of the transaction",
			},
			cli.StringFlag{
				Name:  "params, p",
				Usage: "invoke contract compiler contract params",
			},
			cli.StringFlag{
				Name:  "codeHash, a",
				Usage: "invoke contract compiler contract code hash",
			},
		},
		Action: contractAction,
		OnUsageError: func(c *cli.Context, err error, subCommand bool) error {
			return cli.NewExitError(err, 1)
		},
	}
}