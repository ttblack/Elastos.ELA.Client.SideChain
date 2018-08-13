package info

import (
	"fmt"
	"strconv"

	"github.com/elastos/Elastos.ELA.Client.SideChain/rpc"

	"github.com/urfave/cli"
	"encoding/json"
	"bytes"
)

func infoAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	if c.Bool("connections") {
		result, err := rpc.Call("getconnectioncount", nil)
		if err != nil {
			fmt.Println("error: get node connections failed,", err)
			return err
		}
		printFormat(result)
		return nil
	}

	if c.Bool("neighbor") {
		result, err := rpc.Call("getneighbors", nil)
		if err != nil {
			fmt.Println("error: get node neighbors info failed,", err)
			return err
		}
		printFormat(result)
		return nil
	}

	if c.Bool("state") {
		result, err := rpc.Call("getnodestate", nil)
		if err != nil {
			fmt.Println("error: get node state info failed,", err)
			return err
		}
		printFormat(result)
		return nil
	}

	if c.Bool("currentheight") {
		result, err := rpc.Call("getcurrentheight", nil)
		if err != nil {
			fmt.Println("error: get block count failed,", err)
			return err
		}
		printFormat(result)
		return nil
	}

	if c.Bool("getbestblockhash") {
		result, err := rpc.Call("getbestblockhash", nil)
		if err != nil {
			fmt.Println("error: get best block hash failed,", err)
			return err
		}
		printFormat(result)
		return nil
	}

	if index := c.Int64("getblockhash"); index >= 0 {
		result, err := rpc.Call("getblockhash", rpc.Param("height", index))
		if err != nil {
			fmt.Println("error: get block hash failed,", err)
			return err
		}
		printFormat(result)
		return nil
	}

	if param := c.String("getblock"); param != "" {
		index, err := strconv.ParseInt(param, 10, 64)
		if err == nil {
			result, err := rpc.CallAndUnmarshal("getblockhash", rpc.Param("height", index))
			if err != nil {
				fmt.Println("error: get block failed,", err)
				return err
			}
			param = result.(string)
		}
		result, err := rpc.Call("getblock",
			rpc.Param("blockhash", param).Add("verbosity", 2))
		if err != nil {
			fmt.Println("error: get block failed,", err)
			return err
		}
		printFormat(result)
		return nil
	}

	if param := c.String("gettransaction"); param != "" {
		result, err := rpc.Call("getrawtransaction", rpc.Param("txid", param))
		if err != nil {
			fmt.Println("error: get transaction failed,", err)
			return err
		}
		printFormat(result)
		return nil
	}

	if c.Bool("showtxpool") {
		result, err := rpc.Call("getrawmempool", nil)
		if err != nil {
			fmt.Println("error: get transaction pool failed,", err)
			return err
		}
		printFormat(result)
		return nil
	}

	return nil
}

func printFormat(data []byte) {
	buf := new(bytes.Buffer)
	json.Indent(buf, data, "", "    ")
	fmt.Println(string(buf.Bytes()))
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "info",
		Usage:       "show node information",
		Description: "With side-cli info, you could look up node status, query blocks, transactions, etc.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "connections",
				Usage: "see how many peers are connected with current node",
			},
			cli.BoolFlag{
				Name:  "neighbor, nbr",
				Usage: "show neighbor nodes information",
			},
			cli.BoolFlag{
				Name:  "state",
				Usage: "show current node status",
			},
			cli.BoolFlag{
				Name:  "currentheight, height",
				Usage: "show blockchain height on current node",
			},
			cli.BoolFlag{
				Name:  "getbestblockhash",
				Usage: "show best block hash",
			},
			cli.Int64Flag{
				Name:  "getblockhash, blockh",
				Usage: "query a block's hash with it's height",
				Value: -1,
			},
			cli.StringFlag{
				Name:  "getblock, block",
				Usage: "query a block with height or it's hash",
			},
			cli.StringFlag{
				Name:  "gettransaction, tx",
				Usage: "query a transaction with it's hash",
			},
			cli.BoolFlag{
				Name:  "showtxpool, txpool",
				Usage: "show transactions in node's transaction pool",
			},
		},
		Action: infoAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			return cli.NewExitError(err, 1)
		},
	}
}
