package rpc

import (
	"encoding/json"
	"errors"
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/elastos/Elastos.ELA.Client.SideChain/config"

	"github.com/elastos/Elastos.ELA/common"
)

var (
	RpcUser     = ""
	RpcPassword = ""
)

// Request represent the standard JSON-RPC request data structure.
type Request struct {
	Id      interface{} `json:"id"`
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type Response struct {
	ID      int64  `json:"id"`
	Version string `json:"jsonrpc"`
	*Error  `json:"error"`
	Result  interface{} `json:"result"`
}

type Error struct {
	ID      int64  `json:"id"`
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

var url string

func GetChainHeight() (uint32, error) {
	result, err := CallAndUnmarshal("getblockcount", nil)
	if err != nil {
		return 0, err
	}
	return uint32(result.(float64))-1, nil
}

func GetBlockHash(height uint32) (*common.Uint256, error) {
	result, err := CallAndUnmarshal("getblockhash", Param("height", height))
	if err != nil {
		return nil, err
	}

	hashBytes, err := common.HexStringToBytes(result.(string))
	if err != nil {
		return nil, err
	}
	return common.Uint256FromBytes(hashBytes)
}

func GetBlock(hash *common.Uint256) (*BlockInfo, error) {
	resp, err := CallAndUnmarshal("getblock",
		Param("blockhash", hash.String()).Add("verbosity", 2))
	if err != nil {
		return nil, err
	}
	block := &BlockInfo{}
	unmarshal(&resp, block)

	return block, nil
}

// Call is a util method to send a JSON-RPC request to server.
func Call(method string, params map[string]interface{}) ([]byte, error) {
	if url == "" {
		url = "http://" + config.Params().Host
	}

	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(RpcUser, RpcPassword)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func CallAndUnmarshal(method string, params map[string]interface{}) (interface{}, error) {
	body, err := Call(method, params)
	if err != nil {
		return nil, err
	}

	var resp Response
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return string(body), err
	}

	if resp.Error != nil {
		return nil, errors.New(resp.Error.Message)
	}

	return resp.Result, nil
}

func unmarshal(result interface{}, target interface{}) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, target)
	if err != nil {
		return err
	}
	return nil
}
