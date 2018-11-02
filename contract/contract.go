package contract

import (
	"github.com/urfave/cli"
	"errors"
	"github.com/elastos/Elastos.ELA.SideChain/contract"
	"io/ioutil"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
)

func CreateContractAddress(context *cli.Context) (*contract.Contract, error)  {
	avm := context.String("file")
	if avm == "" {
		return nil, errors.New("lose avm file param")
	}
	code, err := ioutil.ReadFile(avm)
	if err != nil {
		return nil, err
	}
	paramsStr := context.String("params")
	params, err := common.HexStringToBytes(paramsStr)
	if err != nil {
		return nil, err
	}
	//code = append(code, common.SMARTCONTRACT)
	programHash, err := crypto.ToProgramHash(code)

	contract := &contract.Contract{
		Code:        code,
		Parameters:  contract.ByteToContractParameterType(params),
		ProgramHash: *programHash,
	}
	return contract, nil

}
