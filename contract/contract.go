package contract

import (
	"github.com/urfave/cli"
	"errors"
	"io/ioutil"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/elastos/Elastos.ELA.SideChain.NeoVM/contract"
	"github.com/elastos/Elastos.ELA.SideChain.NeoVM/params"
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
	param, err := common.HexStringToBytes(paramsStr)
	if err != nil {
		return nil, err
	}
	programHash, err := params.ToCodeHash(code)

	contract := &contract.Contract{
		Code:        code,
		Parameters:  contract.ByteToContractParameterType(param),
		ProgramHash: *programHash,
	}
	return contract, nil
}

func GetSignStatus(code, param []byte) (haveSign, needSign int, err error) {
	haveSigned, needSigned, _ := crypto.GetSignStatus(code, param)
	if haveSigned > 0 || needSigned > 0 {
		return haveSigned, needSigned, nil
	}

	if len(param) <= 0 && len(code) > 0 {
		return 0, 1, nil
	} else if len(code) > 0 && len(param) > 0 {
		return 0, 0, nil
	}

	return -1, -1, errors.New("invalid redeem script type")
}

func GetScriptType(script []byte) (byte, error) {
	if len(script) <= 0 {
		return 0, errors.New("invalid transaction type, redeem script not a standard or multi sign type")
	}
	return script[len(script)-1], nil
}
