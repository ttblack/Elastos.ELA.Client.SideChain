package wallet

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"encoding/json"
	"io/ioutil"

	"github.com/elastos/Elastos.ELA.Client.SideChain/config"
	"github.com/elastos/Elastos.ELA.Client.SideChain/log"
	"github.com/elastos/Elastos.ELA.Client.SideChain/rpc"
	walt "github.com/elastos/Elastos.ELA.Client.SideChain/wallet"
	. "github.com/elastos/Elastos.ELA.SideChain/core"
	"github.com/elastos/Elastos.ELA.SideChain/contract"
	"github.com/elastos/Elastos.ELA.SideChain/vm"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/urfave/cli"
)

func createSmartContractTransaction(c *cli.Context, wallet walt.Wallet, fee *Fixed64) error {

	deploy := c.Bool("deploy")
	invoke := c.Bool("invoke")

	if deploy && invoke {
		return errors.New("I don't know what to do when both with --deploy and --invoke")
	}

	if (deploy) {
		return createDeployTransaction(c, wallet, fee)
	} else if (invoke) {
		return CreateInvokeTransaction(c, wallet, fee)
	}

	return errors.New("Create smart contract tx with invalid parameters")
}

func createTransaction(c *cli.Context, wallet walt.Wallet) error {

	feeStr := c.String("fee")
	if feeStr == "" {
		return errors.New("use --fee to specify transfer fee")
	}

	fee, err := StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	if c.Bool("deploy") || c.Bool("invoke") {
		return createSmartContractTransaction(c, wallet, fee)
	}

	from := c.String("from")
	if from == "" {
		from, err = SelectAccount(wallet)
		if err != nil {
			return err
		}
	}

	multiOutput := c.String("file")
	if multiOutput != "" {
		return createMultiOutputTransaction(c, wallet, multiOutput, from, fee)
	}

	amountStr := c.String("amount")
	if amountStr == "" {
		return errors.New("use --amount to specify transfer amount")
	}

	amount, err := StringToFixed64(amountStr)
	if err != nil {
		return errors.New("invalid transaction amount")
	}

	var txn *Transaction
	var to string
	standard := c.String("to")
	deposit := c.String("deposit")
	withdraw := c.String("withdraw")

	if deposit != "" {
		to = config.Params().DepositAddress
		txn, err = wallet.CreateCrossChainTransaction(from, to, deposit, amount, fee)
		if err != nil {
			return errors.New("create transaction failed: " + err.Error())
		}
	} else if withdraw != "" {
		to = walt.DESTROY_ADDRESS
		txn, err = wallet.CreateCrossChainTransaction(from, to, withdraw, amount, fee)
		if err != nil {
			return errors.New("create transaction failed: " + err.Error())
		}
	} else if standard != "" {
		to = standard
		lockStr := c.String("lock")
		if lockStr == "" {
			txn, err = wallet.CreateTransaction(from, to, amount, fee)
			if err != nil {
				return errors.New("create transaction failed: " + err.Error())
			}
		} else {
			lock, err := strconv.ParseUint(lockStr, 10, 32)
			if err != nil {
				return errors.New("invalid lock height")
			}
			txn, err = wallet.CreateLockedTransaction(from, to, amount, fee, uint32(lock))
			if err != nil {
				return errors.New("create transaction failed: " + err.Error())
			}
		}
	} else {
		return errors.New("use --to or --deposit or --withdraw to specify receiver address")
	}

	output(0, 0, txn)

	return nil
}

func createMultiOutputTransaction(c *cli.Context, wallet walt.Wallet, path, from string, fee *Fixed64) error {
	if _, err := os.Stat(path); err != nil {
		return errors.New("invalid multi output file path")
	}
	file, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return errors.New("open multi output file failed")
	}

	scanner := bufio.NewScanner(file)
	var multiOutput []*walt.Transfer
	for scanner.Scan() {
		columns := strings.Split(scanner.Text(), ",")
		if len(columns) < 2 {
			return errors.New(fmt.Sprint("invalid multi output line:", columns))
		}
		amountStr := strings.TrimSpace(columns[1])
		amount, err := StringToFixed64(amountStr)
		if err != nil {
			return errors.New("invalid multi output transaction amount: " + amountStr)
		}
		address := strings.TrimSpace(columns[0])
		multiOutput = append(multiOutput, &walt.Transfer{address, amount})
		log.Trace("Multi output address:", address, ", amount:", amountStr)
	}

	lockStr := c.String("lock")
	var txn *Transaction
	if lockStr == "" {
		txn, err = wallet.CreateMultiOutputTransaction(from, fee, multiOutput...)
		if err != nil {
			return errors.New("create multi output transaction failed: " + err.Error())
		}
	} else {
		lock, err := strconv.ParseUint(lockStr, 10, 32)
		if err != nil {
			return errors.New("invalid lock height")
		}
		txn, err = wallet.CreateLockedMultiOutputTransaction(from, fee, uint32(lock), multiOutput...)
		if err != nil {
			return errors.New("create multi output transaction failed: " + err.Error())
		}
	}

	output(0, 0, txn)

	return nil
}

func createDeployTransaction(c *cli.Context, wallet walt.Wallet, fee *Fixed64) error {
	var err error
	from := c.String("from")
	if from == "" {
		from, err = SelectAccount(wallet)
		if err != nil {
			return err
		}
	}

	code := []byte{}

	codeString := c.String("hex")
	avm := c.String("avm")
	if codeString != "" {
		code, err = HexStringToBytes(codeString)
		if err != nil {
			return err
		}
	} else if avm != "" {
		code, err = ioutil.ReadFile(avm)
		if err != nil {
			return err
		}
	} else {
		return errors.New("Create deploy tx should with --hex <code hex> or --avm <avm file> parameter")
	}

	if len(code) == 0 {
		return errors.New("Invalid code with --hex <code>")
	}

	params := make([]string, 0)
	paramsStr := c.String("params")
	if paramsStr != "" {
		err = json.Unmarshal([]byte(paramsStr), &params)
		if err != nil {
			return errors.New("Invalid format with --params <parameter type json>")
		}
	}

	paramTypes := []byte{}
	for _, v := range params {
		if paramType, ok := contract.ParameterTypeMap[v]; ok {
			paramTypes = append(paramTypes, byte(paramType))
		} else {
			return errors.New(fmt.Sprint("Unsupport parameter type: \"", v, "\""))
		}
	}

	returnTypeString := c.String("returntype")
	returnType, ok := contract.ParameterTypeMap[returnTypeString]
	if !ok {
		return errors.New(fmt.Sprint("Unsupport return type: \"", returnTypeString, "\""))
	}

	messageJson := c.String("msg")
	message := make(map[string]string, 0)
	if messageJson != "" {
		err = json.Unmarshal([]byte(messageJson), &message)
		if err != nil {
			return errors.New("Invalid args --msg <message json>")
		}
	}

	code = append(code, SMARTCONTRACT)
	txn, err := wallet.CreateDeployTransaction(from, code, paramTypes, byte(returnType), message, fee)

	// this code is generate a contractAddress when deployTransaction
	programHash, err := crypto.ToProgramHash(code)

	contract := &contract.Contract{
		Code:        code,
		Parameters:  contract.ByteToContractParameterType(paramTypes),
		ProgramHash: *programHash,
	}
	wallet.AddContractAddress(*contract)
	addrs, err := wallet.GetAddresses()

	ShowAccounts(addrs, &contract.ProgramHash, wallet)

	return output(0, 0, txn);
}

func CreateInvokeTransaction(c *cli.Context, wallet walt.Wallet, fee *Fixed64) error {
	var err error
	program := []byte{}
	paramsString := c.String("params")
	if paramsString != "" {

		builder := vm.NewParamsBuider(new(bytes.Buffer))
		params := make([]map[string]interface{}, 0)
		fmt.Println(paramsString)
		err = json.Unmarshal([]byte(paramsString), &params)
		if err != nil {
			return errors.New("Invalid format with --params <parameter type json>")
		}
		for _, v := range params {
			if len(v) != 1 {
				return errors.New("Invalid --params <parameter json>")
			}
			for paramType, paramValue := range v {
				pt := contract.ParameterTypeMap[paramType]
				switch pt {
				case contract.Boolean:
					builder.EmitPushBool(paramValue.(bool))
				case contract.Integer:
					value := paramValue.(float64)
					builder.EmitPushInteger(int64(value))
				case contract.String:
					builder.EmitPushByteArray([]byte(paramValue.(string)))
				case contract.ByteArray, contract.Hash256, contract.Hash160:
					paramBytes, err := HexStringToBytes(paramValue.(string))
					if err != nil {
						return errors.New(fmt.Sprint("Invalid param \"", paramType, "\": ", paramValue))
					}
					builder.EmitPushByteArray(paramBytes)
				}
			}
		}
		program = append(program, builder.Bytes()...)
		if len(program) == 0 {
			return errors.New("Invalid --params <parameter json>")
		}
	}

	codeHashStr := c.String("hex")
	if codeHashStr == "" {
		return errors.New("Missing args --hex <code hash>")
	}

	var codeHashBytes []byte
	codeHashBytes, err = HexStringToBytes(codeHashStr)
	if err != nil {
		return errors.New("convert code error")
	}

	program = append(program, vm.TAILCALL)
	program = append(program, codeHashBytes...)

	codeHash, err := Uint168FromBytes(codeHashBytes)
	if err != nil {
		return err
	}

	from := c.String("from")
	if from == "" {
		from, err = SelectAccount(wallet)
		if err != nil {
			return err
		}
	}

	amountStr := c.String("amount")
	if amountStr == "" {
		return errors.New("use --amount to specify transfer amount")
	}
	amount, err := StringToFixed64(amountStr)
	if err != nil {
		return errors.New("invalid transaction amount")
	}

	to := c.String("to")

	txn, err := wallet.CreateInvokeTransaction(from, to, amount, program, codeHash, fee)
	if err != nil {
		return errors.New("Create invoke tx error")
	}

	return output(0, 0, txn);
}

func signTransaction(name string, password []byte, context *cli.Context, wallet walt.Wallet) error {
	defer ClearBytes(password)

	content, err := getTransactionContent(context)
	if err != nil {
		return err
	}
	rawData, err := HexStringToBytes(content)
	if err != nil {
		return errors.New("decode transaction content failed")
	}

	var txn Transaction
	err = txn.Deserialize(bytes.NewReader(rawData))
	if err != nil {
		return errors.New("deserialize transaction failed")
	}

	program := txn.Programs[0]

	haveSign, needSign, err := crypto.GetSignStatus(program.Code, program.Parameter)
	if haveSign == needSign {
		return errors.New("transaction was fully signed, no need more sign")
	}

	password, err = GetPassword(password, false)
	if err != nil {
		return err
	}

	_, err = wallet.Sign(name, password, &txn)
	if err != nil {
		return err
	}

	haveSign, needSign, _ = crypto.GetSignStatus(program.Code, program.Parameter)
	signType := program.Code[len(program.Code) - 1]
	if signType == SMARTCONTRACT {
		haveSign = needSign
	}
	fmt.Println("[", haveSign, "/", needSign, "] Transaction successfully signed")

	output(haveSign, needSign, &txn)

	return nil
}

func sendTransaction(context *cli.Context) error {
	content, err := getTransactionContent(context)
	if err != nil {
		return err
	}

	result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("data", content))
	if err != nil {
		return err
	}
	fmt.Println(result.(string))
	return nil
}

func getTransactionContent(context *cli.Context) (string, error) {

	// If parameter with file path is not empty, read content from file
	if filePath := strings.TrimSpace(context.String("file")); filePath != "" {

		if _, err := os.Stat(filePath); err != nil {
			return "", errors.New("invalid transaction file path")
		}
		file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
		if err != nil {
			return "", errors.New("open transaction file failed")
		}
		rawData, err := ioutil.ReadAll(file)
		if err != nil {
			return "", errors.New("read transaction file failed")
		}

		content := strings.TrimSpace(string(rawData))
		// File content can not by empty
		if content == "" {
			return "", errors.New("transaction file is empty")
		}
		return content, nil
	}

	content := strings.TrimSpace(context.String("hex"))
	// Hex string content can not be empty
	if content == "" {
		return "", errors.New("transaction hex string is empty")
	}

	return content, nil
}

func output(haveSign, needSign int, txn *Transaction) error {
	// Serialise transaction content
	buf := new(bytes.Buffer)
	txn.Serialize(buf)
	content := BytesToHexString(buf.Bytes())

	// Print transaction hex string content to console
	fmt.Println(content)

	// Output to file
	fileName := "to_be_signed" // Create transaction file name

	if haveSign == 0 {
		//	Transaction created do nothing
	} else if needSign > haveSign {
		fileName = fmt.Sprint(fileName, "_", haveSign, "_of_", needSign)
	} else if needSign == haveSign {
		fileName = "ready_to_send"
	}
	fileName = fileName + ".txn"

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	_, err = file.Write([]byte(content))
	if err != nil {
		return err
	}

	// Print output file to console
	fmt.Println("File: ", fileName)

	return nil
}
