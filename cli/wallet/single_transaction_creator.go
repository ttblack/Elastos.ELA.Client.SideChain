package wallet

import (
	"errors"

	"github.com/urfave/cli"

	core_wallect "github.com/elastos/Elastos.ELA.Client.Core/cli/wallet"
	walt "github.com/elastos/Elastos.ELA.Client.Core/wallet"
	"github.com/elastos/Elastos.ELA.Client.SideChain/common/config"
	tx "github.com/elastos/Elastos.ELA.Utility/core/transaction"
)

type SingleTransactionCreatorSideImpl struct {
	innerCreator *core_wallect.SingleTransactionCreatorImpl
}

func (impl *SingleTransactionCreatorSideImpl) Create(c *cli.Context,
	param *core_wallect.SingleTransactionParameter, wallet walt.Wallet) (*tx.Transaction, error) {
	trans, err := impl.innerCreator.Create(c, param, wallet)
	if trans != nil && err == nil {
		return trans, err
	}

	withdraw := c.String("withdraw")
	if withdraw != "" {
		to := config.Params().DestroyAddress
		return wallet.CreateCrossChainTransaction(param.From, to, withdraw, param.Amount, param.Fee)
	}

	return nil, errors.New("use --to or --deposit or --withdraw to specify receiver address")
}

func init() {
	core_wallect.SingleTransactionCreatorSingleton = &SingleTransactionCreatorSideImpl{
		&core_wallect.SingleTransactionCreatorImpl{}}
}
