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

	"github.com/urfave/cli"

	"github.com/elastos/Elastos.ELA.Utility/crypto"
	. "github.com/elastos/Elastos.ELA.Utility/common"

	"github.com/elastos/Elastos.ELA.SideChain/types"

	"github.com/elastos/Elastos.ELA.SideChain.NeoVM/avm"
	nc "github.com/elastos/Elastos.ELA.SideChain.NeoVM/contract"
	"github.com/elastos/Elastos.ELA.SideChain.NeoVM/params"

	"github.com/elastos/Elastos.ELA.Client.SideChain/config"
	"github.com/elastos/Elastos.ELA.Client.SideChain/log"
	"github.com/elastos/Elastos.ELA.Client.SideChain/rpc"
	walt "github.com/elastos/Elastos.ELA.Client.SideChain/wallet"
	"github.com/elastos/Elastos.ELA.Client.SideChain/contract"
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

	var txn *types.Transaction
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
			spender, err := Uint168FromAddress(from)
			if err != nil {
				return errors.New("create transaction failed: " + err.Error())
			}

			if spender[0] == params.PrefixSmartContract {
				txn, err = createVerificationTransaction(c, wallet, from, to, amount, fee)
			} else {
				txn, err = wallet.CreateTransaction(from, to, amount, fee)
			}
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

func createVerificationTransaction(c *cli.Context, wallet walt.Wallet, from, to string, amount, fee *Fixed64) (*types.Transaction, error) {

	program := []byte{}
	paramsString := c.String("params")
	buffer := new(bytes.Buffer)
	builder := avm.NewParamsBuider(buffer)
	if paramsString != "" {
		err := parseJsonToBytes(paramsString, builder)
		if err != nil {
			return nil, err
		}
		program = append(program, builder.Bytes()...)
		if len(program) == 0 {
			return nil, errors.New("Invalid --params <parameter json>")
		}
	}

	return wallet.CreateTransactionFromContract(from, to, amount, fee, program)
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
	var txn *types.Transaction
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

	param := make([]string, 0)
	paramsStr := c.String("params")
	if paramsStr != "" {
		err = json.Unmarshal([]byte(paramsStr), &param)
		if err != nil {
			return errors.New("Invalid format with --params <parameter type json>")
		}
	}

	paramTypes := []byte{}
	for _, v := range param {
		if paramType, ok := nc.ParameterTypeMap[v]; ok {
			paramTypes = append(paramTypes, byte(paramType))
		} else {
			return errors.New(fmt.Sprint("Unsupport parameter type: \"", v, "\""))
		}
	}

	returnTypeString := c.String("returntype")
	returnType, ok := nc.ParameterTypeMap[returnTypeString]
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
	gasStr := c.String("gas")
	gas, err := StringToFixed64(gasStr)
	if err != nil {
		return err
	}
	if gasStr == "" {
		//deploy is need 490 ela
		value := Fixed64(490.01 * 100000000)
		gas = &value
	}

	//if code[len(code) - 1] != SMARTCONTRACT {
	//	code = append(code, SMARTCONTRACT)
	//}
	txn, err := wallet.CreateDeployTransaction(from, code, paramTypes, byte(returnType), message, fee, gas)
	programHash, err := params.ToCodeHash(code)
	contract := &nc.Contract{
		Code:        code,
		Parameters:  nc.ByteToContractParameterType(paramTypes),
		ProgramHash: *programHash,
	}
	// this code is generate a contractAddress when deployTransaction
	wallet.AddContractAddress(*contract)
	addrs, err := wallet.GetAddresses()

	ShowAccounts(addrs, &contract.ProgramHash, wallet)

	return output(0, 0, txn)
}

func CreateInvokeTransaction(c *cli.Context, wallet walt.Wallet, fee *Fixed64) error {
	var err error
	program := []byte{}
	paramsString := c.String("params")
	buffer := new(bytes.Buffer)
	builder := avm.NewParamsBuider(buffer)
	if paramsString != "" {
		err := parseJsonToBytes(paramsString, builder)
		if err != nil {
			return err
		}
		program = append(program, builder.Bytes()...)
		if len(program) == 0 {
			return errors.New("Invalid --params <parameter json>")
		}
	}

	codeHash := &Uint168{}
	var codeHashBytes []byte
	codeHashStr := c.String("hex")
	avmFile := c.String("avm")
	if codeHashStr != "" {
		codeHashBytes, err = HexStringToBytes(codeHashStr)
		if err != nil {
			return err
		}
		codeHash, err = Uint168FromBytes(codeHashBytes)
		if err == nil {
			program = append(program, avm.TAILCALL)
		} else {
			codeHash = &Uint168{}
		}
		codeHashBytes = params.UInt168ToUInt160(codeHash)
		program = append(program, codeHashBytes...)
	} else if avmFile != "" {
		code, err := ioutil.ReadFile(avmFile)
		if err != nil {
			return err
		}
		program = append(program, code...)
	} else {
		return errors.New("Create invoke tx should with --hex <code hex> or --avm <avm file>")
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

	gasStr := c.String("gas")
	gas, err := StringToFixed64(gasStr)
	if err != nil {
		return err
	}
	txn, err := wallet.CreateInvokeTransaction(from, to, amount, program, codeHash, fee, gas)
	if err != nil {
		return err
	}

	return output(0, 0, txn);
}

func parseJsonToBytes(data string, builder *avm.ParamsBuilder) error {
	params := make([]map[string]interface{}, 0)
	err := json.Unmarshal([]byte(data), &params)
	if err != nil {
		return err
	}
	for i := len(params) - 1; i >= 0; i-- {
	  	v := params[i]
		if len(v) != 1 {
			return errors.New("Invalid --params <parameter json>")
		}
		for paramType, paramValue := range v {
			pt := nc.ParameterTypeMap[paramType]
			switch pt {
			case nc.Boolean:
				builder.EmitPushBool(paramValue.(bool))
			case nc.Integer:
				value := paramValue.(float64)
				builder.EmitPushInteger(int64(value))
			case nc.String:
				builder.EmitPushByteArray([]byte(paramValue.(string)))
			case nc.PublicKey:
				keyBytes, err := HexStringToBytes(strings.TrimSpace(paramValue.(string)))
				if err != nil {
					return err
				}
				_, err = crypto.DecodePoint(keyBytes)
				if err != nil {
					return err
				}
				builder.EmitPushByteArray(keyBytes)
			case nc.ByteArray, nc.Hash256, nc.Hash168, nc.Signature:
				paramBytes, err := HexStringToBytes(paramValue.(string))
				if err != nil {
					return errors.New(fmt.Sprint("Invalid param \"", paramType, "\": ", paramValue))
				}
				builder.EmitPushByteArray(paramBytes)
			case nc.Hash160:
				paramBytes, err := HexStringToBytes(paramValue.(string))
				if err != nil {
					return errors.New(fmt.Sprint("Invalid param \"", paramType, "\": ", paramValue))
				}
				if len(paramBytes) == 21 {
					temp := make([]byte, 20)
					copy(temp, paramBytes[1 :])
					paramBytes = temp
				}
				builder.EmitPushByteArray(paramBytes)
			case nc.Array:
				mjson,_ :=json.Marshal(paramValue)
				count := len(paramValue.([]interface{}))
				err := parseJsonToBytes(string(mjson), builder)
				if err != nil {
					return err
				}
				builder.EmitPushInteger(int64(count))
				builder.Emit(avm.PACK)
			}
		}
	}
	return nil
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

	var txn types.Transaction
	err = txn.Deserialize(bytes.NewReader(rawData))
	if err != nil {
		return errors.New("deserialize transaction failed")
	}

	program := txn.Programs[0]
	haveSign, needSign, err := contract.GetSignStatus(program.Code, program.Parameter)
	if haveSign == needSign  && haveSign != 0{
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

func output(haveSign, needSign int, txn *types.Transaction) error {
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
